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
var argHeartBeat = flag.Duration("overwrite-heartbeat", 1*time.Second, "overwrite all durations between checks")

var initConfig = flag.Bool("init-config", false, "initialize and save a new config file")
var configFile = flag.String("config-file", "go-live-reload.json", "load a config file")
var logLevel = flag.String("log-level", "info", "log level (debug, info, warn, error)")

func usage() {
	println(`Usage: go-live-reload [options]

Note about the --overwrite-heartbeat option:
ParseDuration parses a duration string. A duration string is a possibly signed 
sequence of decimal numbers, each with optional fraction and a unit suffix, 
such as "300ms", "-1.5h" or "2h45m". 

Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

Options:
	`)
	flag.PrintDefaults()
}

func main() {

	flag.Usage = usage
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

	if *argHeartBeat > 0 {
		slog.Info("overwrite-heartbeat", "duration", *argHeartBeat)

		for i := range config.Builds {
			config.Builds[i].HeartBeat = *argHeartBeat
		}
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
