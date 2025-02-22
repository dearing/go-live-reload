package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"runtime/debug"
)

type Config struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Builds      []Build `json:"builds"`
}

func NewConfig() *Config {

	c := &Config{
		Name:        "go-live-reload",
		Description: "A simple live reload server",
		Builds: []Build{
			{
				Name:       "myserver",
				SrcDir:     ".",
				OutDir:     "build",
				BuildArgs:  []string{"build", "-o", "build/myserver"},
				RunCommand: "./build/myserver",
				RunArgs:    []string{"--bind", ":8081"},
				RunWorkDir: "test",
				Globs:      []string{"test/*.go", "test/wwwroot/*"},
				HeartBeat:  *heartBeat,
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
