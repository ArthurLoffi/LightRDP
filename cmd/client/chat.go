package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/pion/webrtc/v5"
)

// Único leitor do stdin no programa.
var stdinLines = sync.OnceValue(func() <-chan string {
	return readLines(os.Stdin)
})

func readLines(r io.Reader) <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			lines <- line
		}
	}()
	return lines
}

// Configura os handlers do DataChannel: o que for digitado é enviado,
// o que chegar é impresso.
func setupChannel(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		fmt.Printf("\n DataChannel '%s' aberto! Digite mensagens: \n", sanitize(dc.Label()))
		go func() {
			for line := range stdinLines() {
				if line = strings.TrimSpace(line); line != "" {
					if err := dc.SendText(line); err != nil {
						fmt.Println("erro ao enviar:", err)
					}
				}
			}
		}()
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("< %s\n", sanitize(string(msg.Data)))
	})
}

// sanitize remove caracteres de controle (sequências de escape ANSI, \r...)
// e de direção bidirecional, para que o outro lado não manipule o terminal.
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
