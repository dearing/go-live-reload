package main

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Build is a struct that represents a build and run process
type Build struct {
	Name          string        `json:"name,omitzero"`
	SrcDir        string        `json:"srcDir,omitzero"`
	OutDir        string        `json:"outDir,omitzero"`
	BuildArgs     []string      `json:"buildArgs,omitzero"`
	BuildEnvirons []string      `json:"buildEnvirons,omitzero"`
	RunCommand    string        `json:"runCommand,omitzero"`
	RunArgs       []string      `json:"runArgs,omitzero"`
	RunEnvirons   []string      `json:"runEnvirons,omitzero"`
	RunWorkDir    string        `json:"runWorkDir,omitzero"`
	Globs         []string      `json:"globs,omitzero"`
	HeartBeat     time.Duration `json:"heartBeat,omitzero"`
}

// Build executes the "go" + BuildArgs command in the SrcDir and return any error.
//
// ex: err := b.Build()
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

// Run executes the configured command with args and environment variables
//
// ex: b.Run(ctx)
func (b *Build) Run(ctx context.Context) {

	slog.Info("build/run start", "name", b.Name, "command", b.RunCommand, "workDir", b.RunWorkDir, "args", b.RunArgs, "environs", b.RunEnvirons)

	cmd := exec.CommandContext(ctx, b.RunCommand, b.RunArgs...)
	cmd.Dir = b.RunWorkDir
	cmd.Env = b.RunEnvirons

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Warn("build/run", "name", b.Name, "error", err)
		return
	}

	slog.Info("build/run completed", "name", b.Name)
}

// Start manages the build and run processes
//
// Calling cancel on the parent context will stop the build and run processes;
// otherwise the restart channel will trigger a rebuild and rerun. If a build
// fails, the routine halts until it receives a signal from the restart channel.
//
// ex: b.Start(parentContext)
func (b *Build) Start(parentContext context.Context, restart chan struct{}) {

	slog.Info("build/watch start", "name", b.Name)

	for {

		err := b.Build()
		if err != nil {
			slog.Error("build/watch", "name", b.Name, "error", err)
			<-restart // block until the watcher says something changed
		}

		runContext, runCancel := context.WithCancel(parentContext)
		go b.Run(runContext)

		select {
		case <-parentContext.Done():
			slog.Warn("build/watch parent interrupt", "name", b.Name)
			runCancel()
			return
		case <-restart:
			slog.Warn("build/watch watcher interrupt", "name", b.Name)
			runCancel()
			continue
		}
	}
}

// Watch starts a ticker and compares scans for changes in the files.
//
// Calling cancel on the parent context will stop the watch process otherwise
// it ticks ever duration to check for changes. If a change is detected it
// signals the restart channel.
//
// ex: b.Watch(ctx)
func (b *Build) Watch(parentContext context.Context, restart chan struct{}) {

	tick := time.NewTicker(b.HeartBeat)
	defer tick.Stop()

	memoized := CheckFiles(b.Globs)

	for {

		select {
		case <-parentContext.Done():
			slog.Error("build/watch parent interrupt", "name", b.Name)
			return
		case <-tick.C:

			start := time.Now()
			files := CheckFiles(b.Globs)

			if len(files) == 0 {
				slog.Warn("build/watch no files found", "name", b.Name)
				continue
			}

			if len(memoized) == 0 {
				slog.Warn("build/watch no files found", "name", b.Name)
				continue
			}

			if len(files) != len(memoized) {
				slog.Debug("build/watch change detected", "name", b.Name, "duration", time.Since(start))
				restart <- struct{}{}
				memoized = files
				continue
			}

			for i, file := range files {
				if file.ModTime() != memoized[i].ModTime() {
					slog.Debug("build/watch change detected", "name", b.Name, "duration", time.Since(start))
					restart <- struct{}{}
					memoized = files
					continue
				}
			}
		}
	}
}

// CheckFiles is a function that takes a list of globs and returns a list of FileInfo
//
//	ex: files := CheckFiles([]string{"test/*.go", "test/wwwroot/*"})
func CheckFiles(globs []string) []fs.FileInfo {
	files := []fs.FileInfo{}

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

			files = append(files, file)
		}

	}

	return files
}
