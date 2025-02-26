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
	Description   string        `json:"description,omitzero"`
	BuildCommand  string        `json:"buildCommand,omitzero"`
	BuildArgs     []string      `json:"buildArgs,omitzero"`
	BuildEnvirons []string      `json:"buildEnvirons,omitzero"`
	BuildWorkDir  string        `json:"buildWorkDir,omitzero"`
	RunCommand    string        `json:"runCommand,omitzero"`
	RunArgs       []string      `json:"runArgs,omitzero"`
	RunEnvirons   []string      `json:"runEnvirons,omitzero"`
	RunWorkDir    string        `json:"runWorkDir,omitzero"`
	Match         []string      `json:"match,omitzero"`
	HeartBeat     time.Duration `json:"heartBeat,omitzero"`
}

// Build executes the "go" + BuildArgs command in the SrcDir and return any error.
//
// ex: err := b.Build()
func (b *Build) Build() error {

	slog.Info("build execute", "name", b.Name, "buildWorkDir", b.BuildWorkDir, "buildCommand", b.BuildCommand, "buildArgs", b.BuildArgs, "buildEnvirons", b.BuildEnvirons)

	start := time.Now()

	cmd := exec.Command(b.BuildCommand, b.BuildArgs...)

	cmd.Dir = b.BuildWorkDir
	cmd.Env = b.BuildEnvirons

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("build", "name", b.Name, "error", err)
		return err
	}

	slog.Info("build success", "name", b.Name, "duration", time.Since(start))
	return nil
}

// Run executes the configured command with args and environment variables
//
// ex: b.Run(ctx)
func (b *Build) Run(ctx context.Context) {

	slog.Info("run execute", "name", b.Name, "runWorkDir", b.RunWorkDir, "runCommand", b.RunCommand, "runArgs", b.RunArgs, "runEnvirons", b.RunEnvirons)

	cmd := exec.CommandContext(ctx, b.RunCommand, b.RunArgs...)
	cmd.Dir = b.RunWorkDir
	cmd.Env = b.RunEnvirons

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Warn("run", "name", b.Name, "error", err)
		return
	}

	slog.Info("run success", "name", b.Name)
}

// Start manages the build and run processes
//
// Calling cancel on the parent context will stop the build and run processes;
// otherwise the restart channel will trigger a rebuild and rerun. If a build
// fails, the routine halts until it receives a signal from the restart channel.
//
// ex: b.Start(parentContext)
func (b *Build) Start(parentContext context.Context, restart chan struct{}) {

	slog.Info("watch start", "name", b.Name, "match", b.Match)

	for {

		err := b.Build()
		if err != nil {
			slog.Error("watch", "name", b.Name, "error", err)
			<-restart // block until the watcher says something changed
		}

		runContext, runCancel := context.WithCancel(parentContext)
		go b.Run(runContext)

		select {
		case <-parentContext.Done():
			slog.Warn("watch shutdown", "name", b.Name)
			runCancel()
			return
		case <-restart:
			slog.Warn("watch restart", "name", b.Name)
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

	memoized := CheckFiles(b.Match)

	for {

		select {
		case <-parentContext.Done():
			slog.Error("watch parent interrupt", "name", b.Name)
			return
		case <-tick.C:

			start := time.Now()
			files := CheckFiles(b.Match)

			if len(files) == 0 {
				slog.Warn("watch no matches found", "name", b.Name)
				continue
			}

			if len(memoized) == 0 {
				slog.Warn("watch no matches found", "name", b.Name)
				continue
			}

			if len(files) != len(memoized) {
				slog.Debug("watch change detected", "name", b.Name, "duration", time.Since(start))
				restart <- struct{}{}
				memoized = files
				continue
			}

			for i, file := range files {
				if file.ModTime() != memoized[i].ModTime() {
					slog.Debug("watch change detected", "name", b.Name, "duration", time.Since(start))
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
			slog.Error("watch", "error", err)
			continue
		}

		for _, match := range matches {
			slog.Debug("watch", "match", match)

			file, err := os.Stat(match)
			if err != nil {
				slog.Error("watch", "error", err)
				continue
			}

			files = append(files, file)
		}

	}

	return files
}
