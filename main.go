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
var argHeartBeat = flag.Duration("overwrite-heartbeat", 1*time.Second, "temporaryly overwrite all build group heartbeats")

var buildGroups = flag.String("build-groups", "", "comma separated list of build groups to run")

var initConfig = flag.Bool("init-config", false, "initialize and save a new config file")
var configFile = flag.String("config-file", "go-live-reload.json", "load a config file")
var logLevel = flag.String("log-level", "info", "log level (debug, info, warn, error)")

func usage() {
	println(`Usage: go-live-reload [options]

This tool takes a set of build groups and runs them in parallel. Each build group
is defined in the configuration file and contains a set of build and run commands
along with arguments and environment variables. The build group will then watch for 
changes based on the "match" values and restart just itself when a modification
is detected or if a new file is added or removed. This is based comparing the 
current matches to the previous matches every heartbeat duration. If you find the 
tool is restarting too frequently or there is too much IO pressure, you can increase 
the heartbeat duration to reduce the frequency of checks.

Tips:

The --overwrite-heartbeat option is used to temporarily overwrite all build group
heartbeats with the specified duration. This is useful for tweaking the heartbeat
based on the host system's performance. Valid options are those that can be parsed
by Go's time.ParseDuration function. You can observe matches and duration with the
--log-level=debug option.

ex: go-live-reload --overwrite-heartbeat=500ms --log-level=debug

The --build-groups option is used to specify a comma separated list of build groups
to run. If no build groups are specified, all build groups defined in the config
will be ran. If no matches are found, the tool will exit with an error.

ex: go-live-reload --build-groups=frontend,backend

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

	// block until we receive an interrupt signal
	for range chanSig {
		slog.Info("interrupt signal received")
		cancel()
		return
	}
}
