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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// arg handling
			rest := cmd.Flags().Args()
			if len(rest) != 1 {
				cmd.Usage()
				cmd.OutOrStderr().Write([]byte("\n"))
				return errors.Errorf("exactly one .kpkg file must be specified, got %d", len(rest))
			}
			packagePath := rest[0]

			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			test, err := cmd.Flags().GetBool("test")
			if err != nil {
				return err
			}

			// the real work
			p, err := kpkg.Open(packagePath)
			if err != nil {
				return err
			}
			// package must remain open until after extraction
			defer p.Close()

			return p.ExtractAll(ctx, output, test, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolP("test", "t", false, "Test extraction without writing files")

	cmd.Flags().StringP("output", "o", "./extracted", "Output directory for extracted files")

	return cmd
}
