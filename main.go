package main

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"flag"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {

	flag.Parse()

	slog.Info("Hello World")

	err := gatherFile("test")
	if err != nil {
		slog.Error("gatherFile", "error", err)
	}

	errChan := make(chan error)
	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGTERM)

	b := &Build{
		Name:       "github.com/dearing/go-live-reload/myserver",
		ChanError:  errChan,
		SrcDir:     "test",
		OutDir:     "test",
		BuildArgs:  []string{"build", "-o", "myserver"},
		RunCommand: "./myserver",
		RunArgs:    []string{"--bind", ":8081"},
		RunWorkDir: "test",
	}

	ctx, cancel := context.WithCancel(context.Background())

	b.BuildAndRun(ctx)

	for {
		select {

		case err := <-errChan:
			slog.Error("main", "error", err)
		case <-chanSig:
			slog.Info("interrupt signal received")
			cancel()
			return
		}
	}
}

func gatherFile(root string) error {
	err := filepath.WalkDir(root, visit)
	return err
}

func visit(path string, entry fs.DirEntry, err error) error {

	if err != nil {
		slog.Error("visit", "error", err)
		return err
	}

	if entry.IsDir() {
		//slog.Info("visit", "path", path, "name", entry.Name(), "isDir", entry.IsDir())
		return nil
	} else {
		file, err := entry.Info()
		if err != nil {
			slog.Error("visit", "error", err)
			return err
		}

		slog.Info("visit", "path", path, "name", entry.Name(), "size", file.Size(), "modTime", file.ModTime())

		hash := sha1.New()
		hash.Write([]byte(path))

		key := base64.StdEncoding.EncodeToString([]byte(path))
		slog.Info("visit", "key", key)

	}

	return nil
}
