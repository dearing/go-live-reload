package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"

	"github.com/dearing/go-live-reload/core"
)

var argVersion = flag.Bool("version", false, "print debug info and exit")
var argHeartBeat = flag.Duration("overwrite-heartbeat", 0, "temporarily overwrite all build group heartbeats")
var buildGroups = flag.String("build-groups", "", "comma separated list of build groups to run")
var initConfig = flag.Bool("init-config", false, "initialize and save a new config file")
var configFile = flag.String("config-file", "go-live-reload.json", "load a config file")
var logLevel = flag.String("log-level", "info", "log level (debug, info, warn, error)")

var staticServerAddr = flag.String("static-server-addr", "", "start a static file server")
var staticServerDir = flag.String("static-server-dir", "", "directory to serve static files from")

var tlsCertFile = flag.String("tls-cert-file", "", "path to TLS certificate file")
var tlsKeyFile = flag.String("tls-key-file", "", "path to TLS key file")

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

1) The --overwrite-heartbeat option is used to temporarily overwrite all build group
heartbeats with the specified duration. This is useful for tweaking the heartbeat
based on the host system's performance. Valid options are those that can be parsed
by Go's time.ParseDuration function. You can observe matches and duration with the
--log-level=debug option.

ex: go-live-reload --overwrite-heartbeat=500ms --log-level=debug

2) The --build-groups option is used to specify a comma separated list of build groups
to run. If no build groups are specified, all build groups defined in the config
will be ran. If no matches are found, the tool will exit with an error.

ex: go-live-reload --build-groups=frontend,backend

3) The ENV lists are appended to the current environment variables. If you need to
overwrite an environment variable, you can do so by specifying the same key in
the ENV list. If you need to clear the environment, set the value to an empty list.
Clearing and then appending is not supported by this tool.

Options:
	`)
	flag.PrintDefaults()
}

func main() {

	// set our custom usage
	flag.Usage = usage
	flag.Parse()

	// attempt set log level
	slog.SetLogLoggerLevel(ParseLogLevel(*logLevel))

	// if --version is set, print version and exit
	if *argVersion {
		Version()
		return
	}

	// if --init-config is set, create a new config file and exit
	if *initConfig {
		c := core.NewConfig()
		err := c.Save(*configFile)
		if err != nil {
			slog.Error("init-config", "error", err)
			return
		}
		slog.Info("init-config", "config", *configFile)
		return
	}

	config := &core.Config{}

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

	// if static server is defined, start it
	if *staticServerAddr != "" && *staticServerDir != "" {
		slog.Warn("static-server", "addr", *staticServerAddr, "dir", *staticServerDir)
		config.StaticServer.BindAddr = *staticServerAddr
		config.StaticServer.StaticDir = *staticServerDir
	}

	// if tls cert and key are defined, set them
	if *tlsCertFile != "" && *tlsKeyFile != "" {
		slog.Warn("tls", "cert", *tlsCertFile, "key", *tlsKeyFile)
		config.TLSCertFile = *tlsCertFile
		config.TLSKeyFile = *tlsKeyFile
	}

	// start static server if BindAddr is defined
	if config.StaticServer.BindAddr != "" {
		go config.RunStatic()
	}

	// check if reverse proxy is defined
	if len(config.ReverseProxy) > 0 {
		go config.RunProxy()
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
	} else {
		slog.Info("build-groups", "groups", groups)
	}

	slog.Info("ready", "config-file", *configFile)

	// this will be the parent context for our build-groups
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builds := 0 // track our build count
	// iterate over each build group and start the build and watch goroutines
	for _, build := range config.Builds {

		// if groups are defined, skip any that are not in the list
		if len(groups) != 0 && !slices.Contains(groups, build.Name) {
			slog.Warn("skipping", "build-group", build.Name)
			continue
		}

		// start and watch the build group using the coordinating over the 'restart' channel
		restart := make(chan struct{})
		go build.Start(ctx, restart) // start build and run loop for this build group
		go build.Watch(ctx, restart) // watch for changes in this build group

		builds++
	}

	// if no builds are found, exit
	if builds == 0 {
		slog.Error("no builds found", "build-groups", *buildGroups, "config-file", *configFile)
		return
	}

	slog.Info("entering run loop", "build-groups", builds)

	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGTERM)

	// block until we receive an interrupt signal
	for range chanSig {
		slog.Info("interrupt signal received")
		cancel()
		return
	}
}

// version retrieves the build information and logs it
func Version() {
	// seems like a nice place to sneak in some debug information
	info, ok := debug.ReadBuildInfo()
	if ok {
		slog.Info("buildInfo", "main", info.Main.Path, "goVersion", info.GoVersion, "version", info.Main.Version)

		if len(info.Deps) > 0 {
			for _, dep := range info.Deps {
				slog.Info("buildInfo.dep", dep.Path, dep.Version)
			}
		}

		for _, setting := range info.Settings {
			slog.Info("buildInfo.setting", setting.Key, setting.Value)
		}
	}
}

// parseLogLevel converts a string to a slog.Level
func ParseLogLevel(value string) slog.Level {

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
