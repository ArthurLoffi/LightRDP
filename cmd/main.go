package main

import (
	"LightRDP/config"
	"LightRDP/connection"
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
<<<<<<< Updated upstream
	"time"

	"github.com/pion/randutil"
	"github.com/pion/webrtc/v4"
)

func main() {
	// Configuração do servidor auxiliar de conexão entre os dois peers.
	// Para conseguirem se conectar mesmo estando atras de NAT ou firewall.
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"localhost:8080"},
			},
		},
	}

	// Criada o par de conexão de um lado.
	// É uma conexão WebRTC para comunicações peer-to-peer
	// com outro PeerConnection pelo lado do navegador.
	// Adiar para fechar essa conexão assim que a aplicação parar.
	peerCnn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer func() {
		if err := peerCnn.Close(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}()

	// Handler para controlar e exibir mudanças no estado da conexão.
	peerCnn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())

		// Muda o estado para falhar se ficar 30 segundos
		// sem atividade ou algum outro tipo de falha.
		if state == webrtc.PeerConnectionStateFailed {
			fmt.Println("Peer connection failed")
			os.Exit(1)
		}

		if state == webrtc.PeerConnectionStateClosed {
			fmt.Println("Peer connection closed")
			os.Exit(0)
		}
	})

	// Criar um novo data channel e fazer o envio de mensagens aleatórias
	peerCnn.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", dataChannel.Label(), dataChannel.ID())

		dataChannel.OnOpen(func() {
			fmt.Printf("DataChannel %s - %d open. Sending random messages every 5 seconds.", dataChannel.Label(), dataChannel.ID())

			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				message, err := randutil.GenerateCryptoRandomString(15, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
=======

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
>>>>>>> Stashed changes
				if err != nil {
					panic(err)
				}

				fmt.Printf("Sending '%s'\n", message)
				if err := dataChannel.SendText(message); err != nil {
					panic(err)
				}
			}
		})

		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel %s - '%s'\n", dataChannel.Label(), string(msg.Data))
		})
	})

	offer := webrtc.SessionDescription{}
	decode(readUntilNewline(), &offer)

	// Cria a SessionDescription remota
	err = peerCnn.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Cria uma resposta
	answer, err := peerCnn.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Crie um canal bloqueado até que a coleta ICE seja concluída.
	gatherComplete := webrtc.GatheringCompletePromise(peerCnn)

	// Define o LocalDescription e inicia os ouvintes UDP
	err = peerCnn.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Bloqueia até que a coleta ICE esteja concluída, desativando o ICE Trickle.
	// Isso porque só podemos trocar uma mensagem de sinalização em prod
	<-gatherComplete

	// Mostra o output em base64
	fmt.Println(encode(peerCnn.LocalDescription()))

	// Bloqueado para sempre
	select {}
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