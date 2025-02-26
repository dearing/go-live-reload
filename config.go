package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"runtime/debug"
	"time"
)

type Config struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Builds      []Build `json:"builds"`
}

// NewConfig returns a new Config with reasonable defaults
func NewConfig() *Config {

	c := &Config{
		Name:        "github.com/dearing/webserver",
		Description: "sample webserver config",
		Builds: []Build{
			{
				Name:        "webserver",
				Description: "sample webserver",
				Match:       []string{"*.go"},
				HeartBeat:   time.Duration(1 * time.Second),
				BuildCmd:    "go",
				BuildArgs:   []string{"build", "-o", "build/webserver"},
				BuildEnv:    []string{"CGO_ENABLED=0"},
				BuildDir:    ".",
				RunCmd:      "./webserver",
				RunArgs:     []string{"--www-bind", ":8081", "--www-root", "wwwroot"},
				RunEnv:      []string{"WWWBIND=8081", "WWWROOT=wwwroot"},
				RunDir:      "build",
			},
		},
	}
	return c
}

// Save saves a json representation of Config to filename
//
//	ex: myConfig.Save("go-live-reload.json")
func (c *Config) Save(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Load reads filename into a Config struct
//
//	ex: myConfig.Load("go-live-reload.json")
func (c *Config) Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, c)
	if err != nil {
		return err
	}
	return nil
}

// version retrieves the build information and logs it
func version() {
	// seems like a nice place to sneak in some debug information
	info, ok := debug.ReadBuildInfo()
	if ok {
		slog.Info("build info", "main", info.Main.Path, "version", info.Main.Version)
		for _, setting := range info.Settings {
			slog.Info("build info", "key", setting.Key, "value", setting.Value)
		}
	}
}

// parseLogLevel converts a string to a slog.Level
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
