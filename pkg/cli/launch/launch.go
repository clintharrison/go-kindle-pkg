package launch

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch [flags] package-id",
		Short: "Launch an installed package by its ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if len(args) < 1 {
				_ = cmd.Usage()
				_, _ = cmd.OutOrStderr().Write([]byte("\n"))
				return nil
			}
			packageID := args[0]

			err := runLaunchScript(ctx, packageID)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func runLaunchScript(ctx context.Context, packageID string) error {
	// TODO: make a package for the package path layout junk
	scriptPath := filepath.Join(version.BaseDir(), "pkgs", packageID, "launch.sh")
	cmd := exec.CommandContext(ctx, "/bin/sh", "-xl", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	slog.Debug("running launch.sh", "path", scriptPath, "cmd", cmd.String())
	return cmd.Run() //nolint:wrapcheck
}
