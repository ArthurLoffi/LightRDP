// Package server implements the signaling server: registers clients by
// ID and relays messages between them. It knows nothing about WebRTC.
package server

import (
	"errors"
	"log/slog"
	"sync"

	"LightRDP/internal/signaling"
)

var (
	errIDTaken       = errors.New("id already registered")
	errAlreadyNamed  = errors.New("connection already registered")
	errEmptyID       = errors.New("empty id")
	errNotRegistered = errors.New("register before sending messages")
)

// Hub keeps the id -> client map and routes messages.
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
	h.log.Info("client registered", "id", id)

	return nil
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Only removes it if the ID still points to this connection.
	if c.id != "" && h.clients[c.id] == c {
		delete(h.clients, c.id)
		h.log.Info("client removed", "id", c.id)
	}
}

func (h *Hub) lookup(id string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c, ok := h.clients[id]
	return c, ok
}

// route handles a message received from "from".
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

	// Prevents a client from sending messages only the server may send.
	if !signaling.Relayable(msg.Type) {
		from.enqueue(signaling.NewError(from.id, "message type not allowed: "+string(msg.Type)))
		return
	}

	// The server sets the sender: prevents a client from impersonating another.
	msg.From = from.id

	target, ok := h.lookup(msg.To)
	if !ok {
		from.enqueue(signaling.NewError(from.id, "target not found: "+msg.To))
		return
	}

	target.enqueue(msg)
}
