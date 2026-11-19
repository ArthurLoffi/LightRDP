// Package transport is the client side of the connection to the signaling server.
package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
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

// Dial connects to the server (e.g. wss://192.168.0.10:8080/ws) and starts reading.
func Dial(ctx context.Context, serverURL string, tlsConfig *tls.Config) (*Conn, error) {
	if err := checkSecureURL(serverURL); err != nil {
		return nil, err
	}

	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.TLSClientConfig = tlsConfig
	opts := &websocket.DialOptions{HTTPClient: &http.Client{Transport: httpTransport}}

	ws, _, err := websocket.Dial(ctx, serverURL, opts)
	if err != nil {
		return nil, fmt.Errorf("transport: dial %s: %w", serverURL, err)
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

// Send is safe for concurrent calls.
func (c *Conn) Send(ctx context.Context, msg signaling.Message) error {
	if err := wsjson.Write(ctx, c.ws, msg); err != nil {
		return fmt.Errorf("transport: send %s: %w", msg.Type, err)
	}
	return nil
}

// Messages is closed when the connection ends; see Err.
func (c *Conn) Messages() <-chan signaling.Message {
	return c.in
}

// Err returns the reason the connection ended.
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

func checkSecureURL(serverURL string) error {
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("transport: dial %s: %w", serverURL, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme == "ws" || scheme == "http") && !IsLoopbackHost(u.Hostname()) {
		return fmt.Errorf("transport: insecure server URL %s: use wss://", serverURL)
	}
	return nil
}

func IsLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
