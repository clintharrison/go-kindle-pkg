package install

import (
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [flags] example.kpkg",
		Short: "Extract and install a .kpkg file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.Errorf("not implemented")
		},
	}

	return cmd
}
