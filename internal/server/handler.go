package server

import (
	"net/http"

	"github.com/coder/websocket"
)

// ServeHTTP faz o upgrade para WebSocket e mantém a conexão até ela fechar.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("upgrade falhou", "remote", r.RemoteAddr, "err", err)
		return
	}
	defer conn.CloseNow()

	h.log.Debug("nova conexão", "remote", r.RemoteAddr)
	newClient(h, conn).run(r.Context())
}
