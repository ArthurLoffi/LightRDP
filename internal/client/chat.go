package client

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

const safetyCodeConfirmedMessage = "lightrdp:safety-code-confirmed"

type chat struct {
	mu              sync.Mutex
	channel         *webrtc.DataChannel
	channelOpen     bool
	localConfirmed  bool
	remoteConfirmed bool
	open            bool
	openedAt        time.Time
}

func (c *chat) attach(dc *webrtc.DataChannel) {
	c.mu.Lock()
	c.channel = dc
	c.mu.Unlock()

	dc.OnOpen(c.onChannelOpen)
	dc.OnMessage(c.onMessage)
}

func (c *chat) onChannelOpen() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.channelOpen = true
	if c.localConfirmed {
		c.sendConfirmation()
	}
	c.openIfReady()
}

func (c *chat) confirmLocal() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.localConfirmed {
		return
	}
	c.localConfirmed = true
	if c.channelOpen {
		c.sendConfirmation()
	}
	c.openIfReady()
}

func (c *chat) isOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.open
}

func (c *chat) onMessage(msg webrtc.DataChannelMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.remoteConfirmed {
		if msg.IsString && string(msg.Data) == safetyCodeConfirmedMessage {
			c.remoteConfirmed = true
			c.openIfReady()
		}
		return
	}
	if !c.open {
		return
	}
	fmt.Printf("< %s\n", sanitize(string(msg.Data)))
}

func (c *chat) sendConfirmation() {
	if err := c.channel.SendText(safetyCodeConfirmedMessage); err != nil {
		fmt.Println("failed to send:", err)
	}
}

func (c *chat) openIfReady() {
	if c.open || !c.channelOpen || !c.localConfirmed || !c.remoteConfirmed {
		return
	}
	c.open = true
	c.openedAt = time.Now()

	fmt.Printf("\n DataChannel '%s' open! Type messages: \n", sanitize(c.channel.Label()))
	go c.forwardInput(c.channel, c.openedAt)
}

func (c *chat) forwardInput(dc *webrtc.DataChannel, openedAt time.Time) {
	for line := range stdinLines() {
		if line.readAt.Before(openedAt) {
			continue
		}
		if text := strings.TrimSpace(line.text); text != "" {
			if err := dc.SendText(text); err != nil {
				fmt.Println("failed to send:", err)
			}
		}
	}
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
