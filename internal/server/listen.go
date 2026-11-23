package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"LightRDP/internal/transport"
)

type ListenConfig struct {
	Bind     string
	Port     int
	Debug    bool
	CertFile string
	KeyFile  string
}

func Listen(cfg ListenConfig) error {
	addr := net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port))

	log := NewLogger(cfg.Debug)
	srv := NewHTTPServer(addr, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	stopped := make(chan error, 1)
	go func() {
		log.Info("signaling server listening", "addr", addr, "path", "/ws", "tls", cfg.CertFile != "")
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			err = Serve(srv, ln, cfg.CertFile, cfg.KeyFile)
		}
		stopped <- err
	}()

	select {
	case err := <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		log.Error("server stopped", "err", err)
		return &StoppedError{Err: err}
	case <-ctx.Done():
	}

	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	return nil
}

type StoppedError struct {
	Err error
}

func (e *StoppedError) Error() string {
	return "server stopped: " + e.Err.Error()
}

func (e *StoppedError) Unwrap() error {
	return e.Err
}

func NewLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func NewHTTPServer(addr string, log *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/ws", NewHub(log))

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func ValidateListenConfig(bind, certFile, keyFile string) error {
	if (certFile == "") != (keyFile == "") {
		return errors.New("--cert and --key must be used together")
	}
	if certFile == "" && !transport.IsLoopbackHost(bind) {
		return errors.New("--cert and --key are required when --bind is not a loopback address")
	}
	return nil
}

func Serve(srv *http.Server, ln net.Listener, certFile, keyFile string) error {
	if certFile == "" {
		return srv.Serve(ln)
	}
	return srv.ServeTLS(ln, certFile, keyFile)
}
