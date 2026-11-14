package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"LightRDP/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "endereço de escuta (\":8080\" escuta em todas as interfaces)")
	debug := flag.Bool("debug", false, "logs detalhados")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	mux := http.NewServeMux()
	mux.Handle("/ws", server.NewHub(log))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("servidor de sinalização ouvindo", "addr", *addr, "path", "/ws")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("servidor parou", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("encerrando")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
