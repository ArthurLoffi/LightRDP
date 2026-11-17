package server

import (
	"net/http"

	"github.com/coder/websocket"
)

// ServeHTTP upgrades to WebSocket and keeps the connection until it closes.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("upgrade failed", "remote", r.RemoteAddr, "err", err)
		return
	}
	defer conn.CloseNow()

	h.log.Debug("new connection", "remote", r.RemoteAddr)
	newClient(h, conn).run(r.Context())
}
