package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pion/webrtc/v5"
)

// The only stdin reader in the program.
var stdinLines = sync.OnceValue(func() <-chan stdinLine {
	return readLines(os.Stdin)
})

type stdinLine struct {
	text   string
	readAt time.Time
}

func readLines(r io.Reader) <-chan stdinLine {
	lines := make(chan stdinLine)
	go func() {
		defer close(lines)
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			lines <- stdinLine{text: line, readAt: time.Now()}
		}
	}()
	return lines
}

// Sets up the DataChannel handlers: typed text is sent,
// received text is printed.
func setupChannel(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		fmt.Printf("\n DataChannel '%s' open! Type messages: \n", sanitize(dc.Label()))
		go func() {
			for line := range stdinLines() {
				if line := strings.TrimSpace(line.text); line != "" {
					if err := dc.SendText(line); err != nil {
						fmt.Println("failed to send:", err)
					}
				}
			}
		}()
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("< %s\n", sanitize(string(msg.Data)))
	})
}

// sanitize removes control characters (ANSI escape sequences, \r...)
// and bidirectional characters, so the other side cannot manipulate the terminal.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return r
		case unicode.IsControl(r),
			r >= 0x202A && r <= 0x202E,
			r >= 0x2066 && r <= 0x2069:
			return -1
		}
		return r
	}, s)
}
