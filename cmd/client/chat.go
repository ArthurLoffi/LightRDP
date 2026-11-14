package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/pion/webrtc/v5"
)

// Único leitor do stdin no programa.
var stdin = bufio.NewReader(os.Stdin)

// Configura os handlers do DataChannel: o que for digitado é enviado,
// o que chegar é impresso.
func setupChannel(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		fmt.Printf("\n DataChannel '%s' aberto! Digite mensagens: \n", dc.Label())
		go func() {
			for {
				line, err := stdin.ReadString('\n')
				if err != nil {
					return
				}
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
