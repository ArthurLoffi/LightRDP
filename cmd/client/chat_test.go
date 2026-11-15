package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/peer"
)

const (
	helperEnv       = "LIGHTRDP_TEST_HELPER"
	helperServerEnv = "LIGHTRDP_TEST_SERVER"
	helperTimeout   = 30 * time.Second
	chatReply       = "hi\x1b[31m‮!\r"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain text", "olá, mundo", "olá, mundo"},
		{"tab", "a\tb", "a\tb"},
		{"carriage return", "a\rb", "ab"},
		{"newline", "a\nb", "ab"},
		{"ansi escape", "\x1b[31mred\x1b[0m", "[31mred[0m"},
		{"null and bell", "a\x00b\x07c", "abc"},
		{"delete and c1 control", "a\x7fb\u0085c", "abc"},
		{"bidi embedding and override", "a‪b‫c‬d‭e‮f", "abcdef"},
		{"bidi isolates", "a⁦b⁧c⁨d⁩e", "abcde"},
		{"next to bidi range", "  ⁥⁪", "  ⁥⁪"},
		{"empty", "", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitize(c.in); got != c.want {
				t.Fatalf("Expected sanitize(%q) = %q, got: %q", c.in, c.want, got)
			}
		})
	}
}

func TestSetupChannel(t *testing.T) {
	cmd := helperCommand(t, "chat")
	stdinWriter, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Expected stdin pipe, got: %v", err)
	}
	stdout := startHelper(t, cmd)
	stderr := cmd.Stderr.(*bytes.Buffer)

	waitForLine(t, stdout, "DataChannel 'chat' aberto! Digite mensagens:")

	if _, err := io.WriteString(stdinWriter, "  hello  \n\n   \nworld\n"); err != nil {
		t.Fatalf("Expected stdin written, got: %v", err)
	}

	waitForLine(t, stdout, "< "+sanitize(chatReply))

	stdinWriter.Close()
	stopHelper(cmd)

	out := stderr.String()
	if strings.Contains(out, "DATA RACE") {
		t.Fatalf("Expected no data race in helper, got: %s", out)
	}
	want := "received \"hello\"\nreceived \"world\"\n"
	if !strings.HasPrefix(out, want) {
		t.Fatalf("Expected helper to receive exactly %q, got: %q", want, out)
	}
}

func TestHelperProcess(t *testing.T) {
	switch os.Getenv(helperEnv) {
	case "chat":
		runChatHelper(t)
	case "host":
		runSessionHelper(t, "host1", "")
	case "join":
		runSessionHelper(t, "join1", "host1")
	}
}

func runChatHelper(t *testing.T) {
	local, err := peer.New(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("Expected local peer, got: %v", err)
	}
	defer local.Close()

	remote, err := peer.New(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("Expected remote peer, got: %v", err)
	}
	defer remote.Close()

	remote.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			text := string(msg.Data)
			fmt.Fprintf(os.Stderr, "received %q\n", text)
			if text == "world" {
				if err := dc.SendText(chatReply); err != nil {
					fmt.Fprintf(os.Stderr, "send reply: %v\n", err)
				}
			}
		})
	})

	dc, err := local.CreateDataChannel("chat")
	if err != nil {
		t.Fatalf("Expected data channel, got: %v", err)
	}
	setupChannel(dc)

	connectTestPeers(t, local, remote)

	time.Sleep(helperTimeout)
}

func connectTestPeers(t *testing.T, offerer, answerer *peer.Peer) {
	t.Helper()

	offerer.OnICECandidate(func(c webrtc.ICECandidateInit) {
		_ = answerer.AddICECandidate(c)
	})
	answerer.OnICECandidate(func(c webrtc.ICECandidateInit) {
		_ = offerer.AddICECandidate(c)
	})

	offer, err := offerer.CreateOffer()
	if err != nil {
		t.Fatalf("Expected offer, got: %v", err)
	}
	answer, err := answerer.HandleOffer(offer)
	if err != nil {
		t.Fatalf("Expected answer, got: %v", err)
	}
	if err := offerer.HandleAnswer(answer); err != nil {
		t.Fatalf("Expected answer applied, got: %v", err)
	}
}

func helperCommand(t *testing.T, mode string, env ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), append([]string{helperEnv + "=" + mode}, env...)...)
	cmd.Stderr = &bytes.Buffer{}

	return cmd
}

func startHelper(t *testing.T, cmd *exec.Cmd) <-chan string {
	t.Helper()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Expected stdout pipe, got: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Expected helper process started, got: %v", err)
	}
	t.Cleanup(func() { stopHelper(cmd) })

	lines := make(chan string, 128)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	return lines
}

func stopHelper(cmd *exec.Cmd) {
	if cmd.ProcessState != nil {
		return
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func waitForLine(t *testing.T, lines <-chan string, want string) {
	t.Helper()

	deadline := time.After(helperTimeout)
	var seen []string
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("Expected line %q, helper output ended: %q", want, seen)
			}
			if strings.TrimSpace(line) == strings.TrimSpace(want) {
				return
			}
			seen = append(seen, line)
		case <-deadline:
			t.Fatalf("Expected line %q before timeout, got: %q", want, seen)
		}
	}
}
