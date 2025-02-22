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

var duration = flag.Duration("d", 5*time.Second, "duration between checks")

func main() {

	flag.Parse()

	slog.Info("go-live-reload started")

	slog.Info("main", "duration", *duration)

	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGTERM)

	b := &Build{
		Name:       "github.com/dearing/go-live-reload/myserver",
		SrcDir:     "test",
		OutDir:     "test",
		BuildArgs:  []string{"build", "-o", "myserver"},
		RunCommand: "./myserver",
		RunArgs:    []string{"--bind", ":8081"},
		RunWorkDir: "test",
		Globs:      []string{"test/*.go", "test/wwwroot/*"},
		Restart:    make(chan struct{}),
		Duration:   *duration,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	go b.Watch(ctx)

	for range chanSig {
		slog.Info("interrupt signal received")
		cancel()
		return
	}
}
