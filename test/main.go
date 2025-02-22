package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var bind = flag.String("bind", ":8080", "Bind address")

func main() {

	flag.Parse()

	slog.Info("Hello World")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Hello received")
		w.Write([]byte("Hello"))
	})

	http.Handle("/wwwroot/", http.StripPrefix("/wwwroot/", http.FileServer(http.Dir("wwwroot"))))

	go func() {
		http.ListenAndServe(*bind, nil)
		slog.Info("server stopped")
	}()

	slog.Info("http server listening", "bind", *bind)

	<-sigchan
	slog.Info("server shut down")

}
