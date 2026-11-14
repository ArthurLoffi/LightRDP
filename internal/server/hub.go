// Package server implementa o servidor de sinalização: registra clientes por
// ID e repassa mensagens entre eles. Não conhece WebRTC.
package server

import (
	"errors"
	"log/slog"
	"sync"

	"LightRDP/internal/signaling"
)

var (
	errIDTaken       = errors.New("id já registrado")
	errAlreadyNamed  = errors.New("conexão já registrada")
	errEmptyID       = errors.New("id vazio")
	errNotRegistered = errors.New("registre-se antes de enviar mensagens")
)

// Hub mantém o mapa id -> cliente e faz o roteamento das mensagens.
type Hub struct {
	log *slog.Logger

	mu      sync.RWMutex
	clients map[string]*Client
}

func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		log:     log,
		clients: make(map[string]*Client),
	}
}

func (h *Hub) register(c *Client, id string) error {
	if id == "" {
		return errEmptyID
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if c.id != "" {
		return errAlreadyNamed
	}
	if _, ok := h.clients[id]; ok {
		return errIDTaken
	}

	c.id = id
	h.clients[id] = c
	h.log.Info("cliente registrado", "id", id)

	return nil
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Só remove se o ID ainda aponta para esta conexão.
	if c.id != "" && h.clients[c.id] == c {
		delete(h.clients, c.id)
		h.log.Info("cliente removido", "id", c.id)
	}
}

func (h *Hub) lookup(id string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c, ok := h.clients[id]
	return c, ok
}

// route trata uma mensagem recebida de "from".
func (h *Hub) route(from *Client, msg signaling.Message) {
	if msg.Type == signaling.TypeRegister {
		if err := h.register(from, msg.From); err != nil {
			from.enqueue(signaling.NewError(msg.From, err.Error()))
		}
		return
	}

	if from.id == "" {
		from.enqueue(signaling.NewError("", errNotRegistered.Error()))
		return
	}

	// Impede que um cliente emita mensagens que só o servidor pode enviar.
	if !signaling.Relayable(msg.Type) {
		from.enqueue(signaling.NewError(from.id, "tipo de mensagem não permitido: "+string(msg.Type)))
		return
	}

	// O servidor define o remetente: impede que um cliente se passe por outro.
	msg.From = from.id

	target, ok := h.lookup(msg.To)
	if !ok {
		from.enqueue(signaling.NewError(from.id, "destino não encontrado: "+msg.To))
		return
	}

	target.enqueue(msg)
}
