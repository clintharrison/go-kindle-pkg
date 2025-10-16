package main

import (
	"log/slog"
	"os"

	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
)

func initLogger() {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)
}

func main() {
	initLogger()

	baseDir := os.Getenv("HOME") + "/Downloads/"
	if h, _ := os.Hostname(); h == "kindle" {
		baseDir = "/mnt/us/kpm/packages/"
	}
	path := baseDir + "koreader_1.2.0_armhf.kpkg"
	pkg, err := kpkg.New(path)
	if err != nil {
		slog.Error("failed to open kpkg", "error", err, "path", path)
		return
	}
	defer pkg.Close()

	if pkg.Manifest == nil {
		slog.Error("kpkg has no manifest")
		return
	}

	slog.Info("kpkg loaded", "id", pkg.Manifest.ID, "name", pkg.Manifest.Name)
}
