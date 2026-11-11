package main

import (
	"LightRDP/config"
	"LightRDP/connection"
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pion/webrtc/v5"
)

var URLS = []string{"stun:stun.l.google.com:19302"}
const CONNECTION_TYPE = "DataChannel"
var stdin = bufio.NewReader(os.Stdin)

func main() {
	peerConfig := config.PeerConfig{
		ICEServers: []webrtc.ICEServer{
			{URLs: URLS},
		},
	}

	cfg := config.NewPeerConfig(peerConfig)

	p, err := connection.NewPeerConnection(cfg, CONNECTION_TYPE)
	if err != nil {
		panic(err)
	}
	defer p.Close()

	p.PeerHandler()

	mode := flag.String("mode", "", "offer (quem inicia) ou answer (quem responde)")
	flag.Parse()
	if *mode != "offer" && *mode != "answer" {
		fmt.Println("uso: go run . -mode offer|answer")
	}

	if *mode == "offer" {
		// Quem faz a offer cria o DataChannel.
		dc, err := p.PeerCnn.CreateDataChannel("chat", nil)
		if err != nil {
			panic(err)
		}
		setupChannel(dc)

		offer, err := p.PeerCnn.CreateOffer(nil)
		if err != nil {
			panic(err)
		}
		gatherComplete := webrtc.GatheringCompletePromise(p.PeerCnn)
		if err = p.PeerCnn.SetLocalDescription(offer); err != nil {
			panic(err)
		}
		<-gatherComplete

		fmt.Println("\n* OFFER *")
		fmt.Println(encode(p.PeerCnn.LocalDescription()))
		fmt.Println("\nCole a ANSWER:")

		answer := webrtc.SessionDescription{}
		decode(readUntilNewline(), &answer)
		if err = p.PeerCnn.SetRemoteDescription(answer); err != nil {
			panic(err)
		}
	} else {
		// Quem responde recebe o DataChannel criado pelo outro lado.
		p.PeerCnn.OnDataChannel(setupChannel)

		fmt.Println("Cole a OFFER:")
		offer := webrtc.SessionDescription{}
		decode(readUntilNewline(), &offer)
		if err = p.PeerCnn.SetRemoteDescription(offer); err != nil {
			panic(err)
		}

		answer, err := p.PeerCnn.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}
		gatherComplete := webrtc.GatheringCompletePromise(p.PeerCnn)
		if err = p.PeerCnn.SetLocalDescription(answer); err != nil {
			panic(err)
		}
		<-gatherComplete

		fmt.Println("\n* ANSWER *")
		fmt.Println(encode(p.PeerCnn.LocalDescription()))
	}

	select {}
}

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

// Leia o stdin no aguardo de uma nova linha.
func readUntilNewline() (in string) {
	var err error

	r := bufio.NewReader(os.Stdin)
	for {
		in, err = r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}

		if in = strings.TrimSpace(in); len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return
}

// JSON encode + base64 para SessionDescription.
func encode(obj *webrtc.SessionDescription) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode uma base64 e JSON em uma SessionDescription.
func decode(in string, obj *webrtc.SessionDescription) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(b, obj); err != nil {
		panic(err)
	}
}