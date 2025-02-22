package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// func Builder(builds chan *Build) {
// 	for build := range builds {
// 		build.Build()
// 	}
// }

// Build is a struct that represents a build process
type Build struct {
	Name string //b.Name of the build process

	ChanError chan error // error channel for build errors

	SrcDir        string   // source directory scoped for go build
	OutDir        string   // output directory for build artifacts
	BuildArgs     []string // appended flags for the go build command
	BuildEnvirons []string // appended environment variables for the go build command

	RunCommand  string   // command to run the built artifact
	RunArgs     []string // appended arguments for the go run command
	RunEnvirons []string // appended environment variables for the go run command
	RunWorkDir  string   // working directory for the go run command

}

func (b *Build) BuildAndRun(ctx context.Context) {
	b.Build(ctx)
	b.Run(ctx)
}

// Build is a method on the Build struct that executes the go build command
func (b *Build) Build(ctx context.Context) {

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
		//b.ChanError <- fmt.Errorf("%s build error: %v", b.Name, err)
	}

	slog.Info("built", "name", b.Name, "duration", time.Since(start))
}

func (b *Build) Run(ctx context.Context) {

	slog.Info("run", "name", b.Name, "command", b.RunCommand, "workDir", b.RunWorkDir, "args", b.RunArgs, "environs", b.RunEnvirons)

	cmd := exec.CommandContext(ctx, b.RunCommand, b.RunArgs...)
	cmd.Dir = b.RunWorkDir
	cmd.Env = b.RunEnvirons

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("run", "name", b.Name, "error", err)
		//b.ChanError <- fmt.Errorf("%s run error: %v", b.Name, err)
		return
	}

	slog.Info("run completed", "name", b.Name)
}
