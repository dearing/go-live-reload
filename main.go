package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"
)

var argVersion = flag.Bool("version", false, "print debug info and exit")
var argHeartBeat = flag.Duration("overwrite-heartbeat", 1*time.Second, "overwrite all durations between checks")

var buildGroups = flag.String("build-groups", "", "comma separated list of build groups to run")

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

	// set our custom usage
	flag.Usage = usage
	flag.Parse()

	// attempt set log level
	slog.SetLogLoggerLevel(parseLogLevel(*logLevel))

	// if --version is set, print version and exit
	if *argVersion {
		version()
		return
	}

	// if --init-config is set, create a new config file and exit
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

	// if no config file is specified, exit
	if *configFile == "" {
		slog.Error("config-file", "error", "no config file specified")
		return
	}

	// if using the default config file, warn the user
	if *configFile == "go-live-reload.json" {
		slog.Warn("using default", "config-file", *configFile)
	}

	// load config file
	err := config.Load(*configFile)
	if err != nil {
		slog.Error("config-file", "error", err)
		return
	}

	// overwrite all heartBeats if --overwrite-heartbeat is set
	if *argHeartBeat > 0 {
		slog.Warn("overwrite-heartbeat", "duration", *argHeartBeat)

		for i := range config.Builds {
			config.Builds[i].HeartBeat = *argHeartBeat
		}
	}

	var groups []string

	// build list of groups to run
	if *buildGroups != "" {
		groups = strings.Split(*buildGroups, ",")
	}

	// if no groups are defined, default to all
	if len(groups) < 1 {
		slog.Warn("no build-groups defined, defaulting to all")
	}

	slog.Info("ready", "config-file", *configFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builds := 0
	for _, b := range config.Builds {

		if len(groups) != 0 && !slices.Contains(groups, b.Name) {
			slog.Warn("skipping", "build-group", b.Name)
			continue
		}

		restart := make(chan struct{})
		go b.Start(ctx, restart)
		go b.Watch(ctx, restart)

		builds++
	}

	// if no builds are found, exit
	if builds == 0 {
		slog.Error("no builds found", "build-groups", *buildGroups, "config-file", *configFile)
		return
	}

	slog.Info("entering run loop", "count", builds)

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
