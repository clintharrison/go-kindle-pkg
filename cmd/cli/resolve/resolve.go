package resolve

import (
	"fmt"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/resolver"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve [flags] org.kindlemodding.example=1.0.0 koreader=1.2.0",
		Short: "Resolve package requests to .kpkg files",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURLs, err := cmd.Flags().GetStringArray("repo")
			if err != nil {
				return errors.Wrap(err, "failed to get repo URLs")
			}
			cmd.OutOrStdout().Write([]byte("Using packages from repositories:\n"))
			for _, u := range repoURLs {
				cmd.OutOrStdout().Write([]byte("  - " + u + "\n"))
			}

			repo, err := repository.NewFromURLs(repoURLs...)
			if err != nil {
				cmd.OutOrStderr().Write([]byte(fmt.Sprintf("ERROR: Unable to create repositories:\n%v\n", err)))
				return errors.Wrap(err, "failed to create repository from URLs")
			}

			packages, err := repo.FetchPackages(cmd.Context())
			if err != nil {
				cmd.OutOrStderr().Write([]byte(fmt.Sprintf("ERROR: Unable to fetch packages from repositories:\n%v\n", err)))
				return errors.Wrap(err, "failed to fetch packages from repositories")
			}
			cmd.OutOrStdout().Write([]byte(fmt.Sprintf("Loaded %d package\n", len(packages))))

			r := resolver.NewResolverForRepositoryPackages(packages)

			// parse the human-friendly-ish constraints
			constraints, err := constraintsFromArgs(args)
			if err != nil {
				return errors.Wrap(err, "failed to parse package constraints from args")
			}

			result, err := r.Resolve(constraints)
			if err != nil {
				cmd.OutOrStderr().Write([]byte(fmt.Sprintf("ERROR: Unable to resolve packages:\n%v\n", err)))
				return errors.Wrap(err, "failed to resolve packages")
			}

			cmd.OutOrStdout().Write([]byte("Resolved packages:\n"))
			for _, art := range result {
				cmd.OutOrStdout().Write([]byte(fmt.Sprintf("  - %s\n", art)))
			}

			return nil
		},
	}
	cmd.Flags().StringArrayP("repo", "r", []string{}, "Repository URL(s) to use (can be specified multiple times)")

	return cmd
}
