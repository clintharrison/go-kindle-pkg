package resolve

import (
	"fmt"

	"github.com/clintharrison/go-kindle-pkg/pkg/cli/clicommon"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve [flags] org.kindlemodding.example=1.0.0 koreader=1.2.0",
		Short: "Resolve package requests to .kpkg files",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := clicommon.GetInitializedResolver(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to initialize resolver")
			}

			// parse the human-friendly-ish constraints
			constraints, err := constraintsFromArgs(args)
			if err != nil {
				return errors.Wrap(err, "failed to parse package constraints from args")
			}

			result, err := r.Resolve(constraints)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "ERROR: Unable to resolve packages:\n%v\n", err) //nolint:errcheck
				return errors.Wrap(err, "failed to resolve packages")
			}

			cmd.OutOrStdout().Write([]byte("Resolved packages:\n")) //nolint:errcheck
			for _, art := range result {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", art) //nolint:errcheck
			}

			return nil
		},
	}
	return cmd
}
