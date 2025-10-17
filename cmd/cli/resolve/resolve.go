package resolve

import (
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve [flags] org.kindlemodding.example=1.0.0 koreader=1.2.0",
		Short: "Resolve package requests to .kpkg files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.Errorf("not implemented")
		},
	}

	return cmd
}
