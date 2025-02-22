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
	"time"
)

type FileSignature struct {
	Hash         string
	CreationTime time.Time
	Modification time.Time
	Size         int64
}

func (fs *FileSignature) Compare(other *FileSignature) int {
	if fs.Hash != other.Hash {
		return FileModified
	}

	if fs.Size != other.Size {
		return FileModified
	}

	if fs.CreationTime != other.CreationTime {
		return FileModified
	}

	if fs.Modification != other.Modification {
		return FileModified
	}

	return NoOp
}

const (
	NoOp = iota
	FileCreated
	FileDeleted
	FileModified
	DirectoryCreated
	DirectoryDeleted
	DirectoryModified
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

	builds := make(chan *Build, 2)
	//go Builder(builds)

	b := &Build{
		Name:      "github.com/dearing/go-live/reload/test",
		ChanError: errChan,
		SrcDir:    "test",
		OutDir:    "test",
		BuildArgs: []string{"build", "-o", "myserver"},

		RunCommand: "./myserver",
		RunArgs:    []string{"--bind", ":8081"},
		RunWorkDir: "test",
	}

	ctx, cancel := context.WithCancel(context.Background())

	b.Build()
	b.Run(ctx)

	for {
		select {

		case err := <-errChan:
			slog.Error("main", "error", err)
		case <-chanSig:
			slog.Info("shutting down")
			cancel()
			close(builds)
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
