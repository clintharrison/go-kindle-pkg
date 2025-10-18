package extract

import (
	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract [flags] example.kpkg",
		Short: "Extract a .kpkg file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			// arg handling
			rest := cmd.Flags().Args()
			if len(rest) != 1 {
				_ = cmd.Usage()
				_, _ = cmd.OutOrStderr().Write([]byte("\n"))
				return errors.Errorf("exactly one .kpkg file must be specified, got %d", len(rest))
			}
			packagePath := rest[0]

			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return errors.AddStack(err)
			}

			test, err := cmd.Flags().GetBool("test")
			if err != nil {
				return errors.AddStack(err)
			}

			// the real work
			pkg, err := kpkg.Open(packagePath)
			if err != nil {
				return errors.Wrapf(err, "kpkg.Open(%q)", packagePath)
			}
			// package must remain open until after extraction
			defer func() { _ = pkg.Close() }()

			return pkg.ExtractAll(ctx, output, test, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolP("test", "t", false, "Test extraction without writing files")

	cmd.Flags().StringP("output", "o", "./extracted", "Output directory for extracted files")

	return cmd
}
