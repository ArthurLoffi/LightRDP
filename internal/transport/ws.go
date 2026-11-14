// Package transport é o lado cliente da conexão com o servidor de sinalização.
package transport

import (
	"context"
	"fmt"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"LightRDP/internal/signaling"
)

const maxMsgSize = 64 << 10

type Conn struct {
	ws     *websocket.Conn
	in     chan signaling.Message
	cancel context.CancelFunc

	mu  sync.Mutex
	err error
}

// Dial conecta ao servidor (ex.: ws://192.168.0.10:8080/ws) e começa a ler.
func Dial(ctx context.Context, url string) (*Conn, error) {
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("transport: dial %s: %w", url, err)
	}
	ws.SetReadLimit(maxMsgSize)

	readCtx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		ws:     ws,
		in:     make(chan signaling.Message, 16),
		cancel: cancel,
	}
	go c.readLoop(readCtx)

	return c, nil
}

// Send é seguro para chamadas concorrentes.
func (c *Conn) Send(ctx context.Context, msg signaling.Message) error {
	if err := wsjson.Write(ctx, c.ws, msg); err != nil {
		return fmt.Errorf("transport: send %s: %w", msg.Type, err)
	}
	return nil
}

// Messages é fechado quando a conexão termina; veja Err.
func (c *Conn) Messages() <-chan signaling.Message {
	return c.in
}

// Err retorna o motivo do fim da conexão.
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *Conn) Close() error {
	c.cancel()
	return c.ws.Close(websocket.StatusNormalClosure, "")
}

func (c *Conn) readLoop(ctx context.Context) {
	defer close(c.in)

	for {
		var msg signaling.Message
		if err := wsjson.Read(ctx, c.ws, &msg); err != nil {
			c.mu.Lock()
			c.err = err
			c.mu.Unlock()
			return
		}

		select {
		case c.in <- msg:
		case <-ctx.Done():
			return
		}
	}
}
