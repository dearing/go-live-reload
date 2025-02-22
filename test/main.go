package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var bind = flag.String("bind", ":8080", "bind address")

func main() {

	flag.Parse()

	slog.Info("hello world")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("hello received")
		w.Write([]byte("hello world"))
	})

	handler.Handle("/", http.FileServer(http.Dir("wwwroot")))
	server := &http.Server{
		Addr:    *bind,
		Handler: handler,
	}

	go func() {
		slog.Info("http server listening", "bind", *bind)
		server.ListenAndServe()
		slog.Info("http server stopped")
	}()

	<-sigchan

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown", "error", err)
	}
}
