package main

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"
)

// Build is a struct that represents a build process
type Build struct {
	Name          string   //b.Name of the build process
	SrcDir        string   // source directory scoped for go build
	OutDir        string   // output directory for build artifacts
	BuildArgs     []string // appended flags for the go build command
	BuildEnvirons []string // appended environment variables for the go build command
	RunCommand    string   // command to run the built artifact
	RunArgs       []string // appended arguments for the go run command
	RunEnvirons   []string // appended environment variables for the go run command
	RunWorkDir    string   // working directory for the go run command
	Globs         []string // globs to watch for changes
	Memoized      []fs.FileInfo
	Restart       chan struct{}
	Duration      time.Duration
}

// Build is a method on the Build struct that executes the go build command
func (b *Build) Build() error {

	slog.Info("build", "name", b.Name, "srcDir", b.SrcDir, "outDir", b.OutDir, "flags", b.BuildArgs)

	start := time.Now()

	cmd := exec.Command("go", b.BuildArgs...)

	cmd.Dir = b.SrcDir
	cmd.Env = b.BuildEnvirons

	// have the command inehrit our stdout and stderr
	// TODO: consider maybe structuring the output to slog
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("build", "name", b.Name, "error", err)
		return err
	}

	slog.Info("build", "name", b.Name, "duration", time.Since(start))
	return nil
}

func (b *Build) Run(ctx context.Context) {

	slog.Info("build/run start", "name", b.Name, "command", b.RunCommand, "workDir", b.RunWorkDir, "args", b.RunArgs, "environs", b.RunEnvirons)

	cmd := exec.CommandContext(ctx, b.RunCommand, b.RunArgs...)
	cmd.Dir = b.RunWorkDir
	cmd.Env = b.RunEnvirons

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("build/run", "name", b.Name, "error", err)
		return
	}

	slog.Info("build/run completed", "name", b.Name)
}

// Watch is a method on the Build struct that watches for changes in the globs
//
// ex: b.Start(ctx)
func (b *Build) Start(parentContext context.Context) {

	slog.Info("build/watch start", "name", b.Name)

	for {

		err := b.Build()
		if err != nil {
			slog.Error("build/watch", "name", b.Name, "error", err)
			<-b.Restart // block until the watcher says something changed
		}

		b.Memoized = CheckFiles(b.Globs)

		runContext, runCancel := context.WithCancel(parentContext)
		go b.Run(runContext)

		select {
		case <-parentContext.Done():
			slog.Warn("build/watch parent interrupt", "name", b.Name)
			runCancel()
			return
		case <-b.Restart:
			slog.Warn("build/watch watcher interrupt", "name", b.Name)
			runCancel()
			continue
		}
	}
}

// Watch is a method on the Build struct that watches for changes in the globs
//
// ex: b.Watch(ctx)
func (b *Build) Watch(parentContext context.Context) {

	tick := time.NewTicker(*duration)
	defer tick.Stop()

	for {
		select {
		case <-parentContext.Done():
			slog.Error("build/watch parent interrupt", "name", b.Name)
			return
		case <-tick.C:
			start := time.Now()
			files := CheckFiles(b.Globs)

			if reflect.DeepEqual(b.Memoized, files) {
				//slog.Info("build/watch no change detected", "name", b.Name, "duration", time.Since(start))
				continue
			}

			b.Memoized = files
			slog.Debug("build/watch change detected", "name", b.Name, "duration", time.Since(start))
			b.Restart <- struct{}{}
		}
	}
}

// CheckFiles is a function that takes a list of globs and returns a list of FileInfo
//
//	ex: files := CheckFiles([]string{"test/*.go", "test/wwwroot/*"})
func CheckFiles(globs []string) []fs.FileInfo {
	scratch := []fs.FileInfo{}

	for _, glob := range globs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			slog.Error("build/watch", "error", err)
			continue
		}

		for _, match := range matches {
			slog.Debug("build/watch", "match", match)

			file, err := os.Stat(match)
			if err != nil {
				slog.Error("build/watch", "error", err)
				continue
			}

			scratch = append(scratch, file)
		}

	}

	return scratch
}
