package main

import (
	"github.com/clintharrison/go-kindle-pkg/pkg/cli/createkpkg"
	"github.com/clintharrison/go-kindle-pkg/pkg/cli/extract"
	"github.com/clintharrison/go-kindle-pkg/pkg/cli/install"
	"github.com/clintharrison/go-kindle-pkg/pkg/cli/list"
	"github.com/clintharrison/go-kindle-pkg/pkg/cli/resolve"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   version.CLIName,
		Short: "Manage .kpkg packages for Kindle",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		SilenceUsage: true,
	}

	cmd.PersistentFlags().String("install-dir", version.BaseDir+"/pkgs", "Directory for unpacked apps and libraries")
	cmd.PersistentFlags().String("download-dir", version.BaseDir+"/downloads", "Directory to store downloaded .kpkg files")
	cmd.PersistentFlags().StringArrayP("repo", "r", []string{},
		"Repository URL(s) to use (can be specified multiple times)")

	cmd.AddCommand(createkpkg.NewCommand())
	cmd.AddCommand(extract.NewCommand())
	cmd.AddCommand(install.NewCommand())
	cmd.AddCommand(list.NewCommand())
	cmd.AddCommand(resolve.NewCommand())

	return cmd
}
