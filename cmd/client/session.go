package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v5"

	"LightRDP/internal/peer"
	"LightRDP/internal/signaling"
	"LightRDP/internal/transport"
)

const sendTimeout = 5 * time.Second

// session liga a sinalização (transport) à conexão WebRTC (peer).
// Suporta uma sessão P2P por vez.
type session struct {
	ctx    context.Context
	conn   *transport.Conn
	cfg    webrtc.Configuration
	selfID string

	// Definidos antes de qualquer callback do pion disparar.
	remoteID string
	peer     *peer.Peer

	connected atomic.Bool
	done      chan error
}

func newSession(ctx context.Context, conn *transport.Conn, cfg webrtc.Configuration, selfID string) *session {
	return &session{
		ctx:    ctx,
		conn:   conn,
		cfg:    cfg,
		selfID: selfID,
		done:   make(chan error, 1),
	}
}

func (s *session) close() {
	if s.peer != nil {
		_ = s.peer.Close()
	}
}

func (s *session) register() error {
	return s.send(signaling.TypeRegister, "", nil)
}

// join é chamado por quem conecta: prepara o peer e pede a sessão ao host.
func (s *session) join(hostID string) error {
	if err := s.startPeer(hostID); err != nil {
		return err
	}
	// Quem responde recebe o DataChannel criado pelo outro lado.
	s.peer.OnDataChannel(setupChannel)

	return s.send(signaling.TypeConnect, hostID, nil)
}

// loop processa mensagens do servidor até a sessão terminar.
func (s *session) loop() error {
	msgs := s.conn.Messages()

	for {
		select {
		case <-s.ctx.Done():
			return nil

		case err := <-s.done:
			return err

		case msg, ok := <-msgs:
			if !ok {
				// Sem servidor, a conexão P2P já aberta continua funcionando.
				if s.connected.Load() {
					fmt.Println("Servidor de sinalização desconectado; chat P2P continua.")
					msgs = nil
					continue
				}
				return fmt.Errorf("servidor de sinalização desconectado: %w", s.conn.Err())
			}
			if err := s.handle(msg); err != nil {
				return err
			}
		}
	}
}

func (s *session) handle(msg signaling.Message) error {
	switch msg.Type {
	case signaling.TypeConnect:
		return s.onConnect(msg.From)

	case signaling.TypeOffer:
		if !s.fromRemote(msg) {
			return nil
		}
		var offer webrtc.SessionDescription
		if err := msg.Decode(&offer); err != nil {
			return err
		}
		answer, err := s.peer.HandleOffer(offer)
		if err != nil {
			return err
		}
		return s.send(signaling.TypeAnswer, s.remoteID, answer)

	case signaling.TypeAnswer:
		if !s.fromRemote(msg) {
			return nil
		}
		var answer webrtc.SessionDescription
		if err := msg.Decode(&answer); err != nil {
			return err
		}
		return s.peer.HandleAnswer(answer)

	case signaling.TypeCandidate:
		if !s.fromRemote(msg) {
			return nil
		}
		var c webrtc.ICECandidateInit
		if err := msg.Decode(&c); err != nil {
			return err
		}
		return s.peer.AddICECandidate(c)

	case signaling.TypeError:
		var e signaling.ErrorPayload
		_ = msg.Decode(&e)
		// Antes da conexão P2P, erro de sinalização encerra a sessão.
		if !s.connected.Load() {
			return errors.New("servidor: " + e.Message)
		}
		fmt.Println("aviso do servidor:", e.Message)
		return nil

	default:
		fmt.Println("mensagem desconhecida:", msg.Type)
		return nil
	}
}

// onConnect é chamado no host quando alguém pede uma sessão.
func (s *session) onConnect(from string) error {
	if s.peer != nil {
		return s.conn.Send(s.ctx, signaling.NewError(from, "host ocupado"))
	}

	fmt.Printf("Pedido de conexão de %q\n", from)
	if err := s.startPeer(from); err != nil {
		return err
	}

	// Quem faz a offer cria o DataChannel.
	dc, err := s.peer.CreateDataChannel("chat")
	if err != nil {
		return err
	}
	setupChannel(dc)

	offer, err := s.peer.CreateOffer()
	if err != nil {
		return err
	}
	return s.send(signaling.TypeOffer, s.remoteID, offer)
}

func (s *session) startPeer(remoteID string) error {
	p, err := peer.New(s.cfg)
	if err != nil {
		return err
	}

	s.remoteID = remoteID
	s.peer = p

	// Trickle ICE: cada candidato local vai para o outro lado assim que surge.
	p.OnICECandidate(func(c webrtc.ICECandidateInit) {
		if err := s.send(signaling.TypeCandidate, s.remoteID, c); err != nil {
			fmt.Println("erro ao enviar candidato:", err)
		}
	})

	// Handler para controlar e exibir mudanças no estado da conexão.
	p.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			s.connected.Store(true)
		// Muda o estado para falhar se ficar 30 segundos
		// sem atividade ou algum outro tipo de falha.
		case webrtc.PeerConnectionStateFailed:
			s.finish(errors.New("peer connection failed"))
		case webrtc.PeerConnectionStateClosed:
			s.finish(nil)
		}
	})

	return nil
}

// fromRemote descarta mensagens de quem não é o par atual.
func (s *session) fromRemote(msg signaling.Message) bool {
	return s.peer != nil && msg.From == s.remoteID
}

func (s *session) send(t signaling.Type, to string, payload any) error {
	msg, err := signaling.New(t, s.selfID, to, payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(s.ctx, sendTimeout)
	defer cancel()

	return s.conn.Send(ctx, msg)
}

func (s *session) finish(err error) {
	select {
	case s.done <- err:
	default:
	}
}
