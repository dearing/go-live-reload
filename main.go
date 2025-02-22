package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var heartBeat = flag.Duration("d", 5*time.Second, "duration between checks")
var initConfig = flag.Bool("init-config", false, "initialize and save a new config file")
var loadConfig = flag.String("load-config", "go-live-reload.json", "load a config file")

func main() {

	flag.Parse()

	if *initConfig {
		c := NewConfig()
		err := c.Save(*loadConfig)
		if err != nil {
			slog.Error("main/init-config", "error", err)
			return
		}
		slog.Info("main/init-config", "config", *loadConfig)
		return
	}

	config := &Config{}
	err := config.Load(*loadConfig)
	if err != nil {
		slog.Error("main/load-config", "error", err)
		return
	}

	slog.Info("main ready", "load-config", *loadConfig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, b := range config.Builds {
		restart := make(chan struct{})
		go b.Start(ctx, restart)
		go b.Watch(ctx, restart)
	}

	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGTERM)

	for range chanSig {
		slog.Info("interrupt signal received")
		cancel()
		return
	}
}
