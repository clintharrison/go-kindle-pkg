package version

import (
	"log/slog"
	"os"
)

const (
	CLIName     = "kpmgo"
	baseDir     = "/mnt/us/kpm"
	FullVersion = CLIName + " v" + Version
	Version     = "0.0.1"
)

var logged = false

func BaseDir() string {
	hostname, err := os.Hostname()
	if err == nil && hostname == "kindle" {
		return baseDir
	}
	// for non-Kindle testing, use a temp directory
	tmpDir := os.TempDir()
	baseDir := tmpDir + "/kpmbase"
	if !logged {
		logged = true
		slog.Info("Running on non-Kindle device; using temporary base directory", "baseDir", baseDir)
	}
	return baseDir
}

func UserstoreDir() string {
	hostname, err := os.Hostname()
	if err == nil && hostname == "kindle" {
		return "/mnt/us"
	}
	// for non-Kindle testing, use a temp directory
	dir := BaseDir() + "/userstore"
	os.MkdirAll(dir, 0o755)
	return dir
}
