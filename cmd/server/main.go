package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"LightRDP/internal/server"
	"LightRDP/internal/transport"
)

func main() {
	bind := flag.String("bind", "127.0.0.1", "listen address (use 0.0.0.0 to listen all interfaces; non-loopback addresses require -cert and -key)")
	port := flag.Int("port", 8080, "port")
	debug := flag.Bool("debug", false, "verbose logs")
	certFile := flag.String("cert", "", "TLS certificate file (PEM); serves wss:// together with -key")
	keyFile := flag.String("key", "", "TLS private key file (PEM); serves wss:// together with -cert")
	flag.Parse()

	if err := validateListenConfig(*bind, *certFile, *keyFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	addr := net.JoinHostPort(*bind, strconv.Itoa(*port))

	log := newLogger(*debug)
	srv := newServer(addr, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("signaling server listening", "addr", addr, "path", "/ws", "tls", *certFile != "")
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			err = serve(srv, ln, *certFile, *keyFile)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func newServer(addr string, log *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/ws", server.NewHub(log))

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func validateListenConfig(bind, certFile, keyFile string) error {
	if (certFile == "") != (keyFile == "") {
		return errors.New("-cert and -key must be used together")
	}
	if certFile == "" && !transport.IsLoopbackHost(bind) {
		return errors.New("-cert and -key are required when -bind is not a loopback address")
	}
	return nil
}

func serve(srv *http.Server, ln net.Listener, certFile, keyFile string) error {
	if certFile == "" {
		return srv.Serve(ln)
	}
	return srv.ServeTLS(ln, certFile, keyFile)
}
