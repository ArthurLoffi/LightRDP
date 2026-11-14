package server

import (
	"context"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"LightRDP/internal/signaling"
)

const (
	sendBuffer   = 64
	maxMsgSize   = 64 << 10 // SDP com candidatos cabe com folga.
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
)

// Client representa uma conexão WebSocket com o servidor.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan signaling.Message

	// Escrito apenas pelo Hub, sob h.mu.
	id string
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	conn.SetReadLimit(maxMsgSize)

	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan signaling.Message, sendBuffer),
	}
}

// run bloqueia até a conexão terminar.
func (c *Client) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.writePump(ctx)
	c.readPump(ctx)

	c.hub.unregister(c)
}

// enqueue nunca bloqueia: cliente lento perde a mensagem em vez de travar o hub.
func (c *Client) enqueue(msg signaling.Message) {
	select {
	case c.send <- msg:
	default:
		c.hub.log.Warn("fila cheia, mensagem descartada", "id", c.id, "type", msg.Type)
	}
}

func (c *Client) readPump(ctx context.Context) {
	for {
		var msg signaling.Message
		if err := wsjson.Read(ctx, c.conn, &msg); err != nil {
			c.hub.log.Debug("leitura encerrada", "id", c.id, "err", err)
			return
		}
		c.hub.route(c, msg)
	}
}

// writePump é o único lugar que escreve mensagens na conexão.
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.send:
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := wsjson.Write(wctx, c.conn, msg)
			cancel()
			if err != nil {
				c.conn.CloseNow()
				return
			}

		case <-ticker.C:
			// Detecta conexões mortas.
			pctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pctx)
			cancel()
			if err != nil {
				c.conn.CloseNow()
				return
			}
		}
	}
}
