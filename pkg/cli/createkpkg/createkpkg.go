package createkpkg

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-kpkg [flags] <input-directory>",
		Short: "Create a .kpkg file",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "invalid output flag: %v", err)
				return errors.Annotate(err, "invalid output flag")
			}

			// validate input directory
			if len(args) != 1 {
				return errors.New("input directory is required")
			}
			inputDir, err := filepath.Abs(args[0])
			if err != nil {
				return errors.Wrapf(err, "filepath.Abs(%q)", args[0])
			}
			di, err := os.Stat(inputDir)
			if err != nil {
				return errors.Wrapf(err, "os.Stat(%q)", inputDir)
			}
			if !di.IsDir() {
				return errors.Errorf("input path %q must be a directory", inputDir)
			}

			// default to <dirname>.kpkg
			// if the dirname is unknown, use "package.kpkg"
			if output == "" {
				base := path.Base(di.Name())
				if base == "." || base == "/" || base == "" {
					base = "package"
				}
				output = base + ".kpkg"
			}

			return kpkg.Build(cmd.Context(), inputDir, output)
		},
	}
	cmd.Flags().StringP("output", "o", "", "Output .kpkg file path")

	return cmd
}
