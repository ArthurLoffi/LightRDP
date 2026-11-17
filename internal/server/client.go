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
	maxMsgSize   = 64 << 10 // SDP with candidates fits comfortably.
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
)

// Client represents a WebSocket connection to the server.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan signaling.Message

	// Written only by the Hub, under h.mu.
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

// run blocks until the connection ends.
func (c *Client) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.writePump(ctx)
	c.readPump(ctx)

	c.hub.unregister(c)
}

// enqueue never blocks: a slow client loses the message instead of stalling the hub.
func (c *Client) enqueue(msg signaling.Message) {
	select {
	case c.send <- msg:
	default:
		c.hub.log.Warn("queue full, message dropped", "id", c.id, "type", msg.Type)
	}
}

func (c *Client) readPump(ctx context.Context) {
	for {
		var msg signaling.Message
		if err := wsjson.Read(ctx, c.conn, &msg); err != nil {
			c.hub.log.Debug("read ended", "id", c.id, "err", err)
			return
		}
		c.hub.route(c, msg)
	}
}

// writePump is the only place that writes messages to the connection.
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
			// Detects dead connections.
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
