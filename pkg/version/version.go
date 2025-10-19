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

func BaseDir() string {
	hostname, err := os.Hostname()
	if err == nil && hostname == "kindle" {
		return baseDir
	}
	// for non-Kindle testing, use a temp directory
	tmpDir := os.TempDir()
	baseDir := tmpDir + "/kpmbase"
	slog.Info("using temporary base dir for non-Kindle host", "baseDir", baseDir)
	os.RemoveAll(baseDir)
	return baseDir
}
