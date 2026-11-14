package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
		fmt.Printf("< %s\n", string(msg.Data))
	})
}
