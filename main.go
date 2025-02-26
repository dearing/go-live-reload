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

var argVersion = flag.Bool("version", false, "print debug info and exit")

var heartBeat = flag.Duration("heartbeat", 1*time.Second, "duration between checks")
var initConfig = flag.Bool("init-config", false, "initialize and save a new config file")
var configFile = flag.String("config-file", "go-live-reload.json", "load a config file")
var logLevel = flag.String("log-level", "info", "log level (debug, info, warn, error)")

func main() {

	flag.Parse()

	slog.SetLogLoggerLevel(parseLogLevel(*logLevel))

	if *argVersion {
		version()
		return
	}

	if *initConfig {
		c := NewConfig()
		err := c.Save(*configFile)
		if err != nil {
			slog.Error("init-config", "error", err)
			return
		}
		slog.Info("init-config", "config", *configFile)
		return
	}

	config := &Config{}
	err := config.Load(*configFile)
	if err != nil {
		slog.Error("config-file", "error", err)
		return
	}

	slog.Info("ready", "config-file", *configFile)

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

func parseLogLevel(value string) slog.Level {

	switch value {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		slog.Warn("parseLogLevel", "unknown log level", value)
		return slog.LevelDebug
	}
}
