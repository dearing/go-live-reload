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
	Name        string        `json:"name,omitzero"`
	Description string        `json:"description,omitzero"`
	Match       []string      `json:"match,omitzero"`
	HeartBeat   time.Duration `json:"heartBeat,omitzero"`
	BuildCmd    string        `json:"buildCmd,omitzero"`
	BuildArgs   []string      `json:"buildArgs,omitzero"`
	BuildEnv    []string      `json:"buildEnv,omitzero"`
	BuildDir    string        `json:"buildDir,omitzero"`
	RunCmd      string        `json:"runCmd,omitzero"`
	RunArgs     []string      `json:"runArgs,omitzero"`
	RunEnv      []string      `json:"runEnv,omitzero"`
	RunDir      string        `json:"runDir,omitzero"`
}

// Build executes the "go" + BuildArgs command in the SrcDir and return any error.
//
// ex: err := b.Build()
func (b *Build) Build() error {

	slog.Info("build execute", "name", b.Name, "buildDir", b.BuildDir, "buildCmd", b.BuildCmd, "buildArgs", b.BuildArgs, "buildEnv", b.BuildEnv)

	start := time.Now()

	cmd := exec.Command(b.BuildCmd, b.BuildArgs...)

	cmd.Dir = b.BuildDir

	// combine the current process environment with the provided environs
	if b.BuildEnv != nil {
		cmd.Env = append(os.Environ(), b.BuildEnv...)
	}

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

	slog.Info("run execute", "name", b.Name, "runDir", b.RunDir, "runCmd", b.RunCmd, "runArgs", b.RunArgs, "runEnv", b.RunEnv)

	cmd := exec.CommandContext(ctx, b.RunCmd, b.RunArgs...)

	cmd.Dir = b.RunDir

	// combine the current process environment with the provided environs
	if b.RunEnv != nil {
		cmd.Env = append(os.Environ(), b.RunEnv...)
	}

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
			slog.Warn("shutdown signaled", "name", b.Name)
			runCancel()
			return
		case <-restart:
			slog.Warn("restart signal", "name", b.Name)
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

	memoized := MatchFiles(b.Match)

	for {

		select {
		case <-parentContext.Done():
			slog.Error("watch parent interrupt", "name", b.Name)
			return
		case <-tick.C:

			start := time.Now()
			files := MatchFiles(b.Match)

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

// MatchFiles is a function that takes a list of globs and returns array of FileInfo
//
//	ex: files := MatchFiles([]string{"test/*.go", "test/wwwroot/*"})
func MatchFiles(globs []string) []fs.FileInfo {
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
