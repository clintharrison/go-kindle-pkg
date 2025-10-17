package main

import (
	"github.com/clintharrison/go-kindle-pkg/cmd/cli/extract"
	"github.com/clintharrison/go-kindle-pkg/cmd/cli/install"
	"github.com/clintharrison/go-kindle-pkg/cmd/cli/list"
	"github.com/clintharrison/go-kindle-pkg/cmd/cli/resolve"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kpm",
		Short: "Manage kpm packages for Kindle",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		SilenceUsage: true,
	}

	cmd.PersistentFlags().String("install-dir", "/mnt/us/kpm/pkgs", "Directory for unpacked apps and libraries")
	cmd.PersistentFlags().String("download-dir", "/mnt/us/kpm/downloads", "Directory to store downloaded .kpkg files")

	cmd.AddCommand(extract.NewCommand())
	cmd.AddCommand(install.NewCommand())
	cmd.AddCommand(list.NewCommand())
	cmd.AddCommand(resolve.NewCommand())

	return cmd
}
