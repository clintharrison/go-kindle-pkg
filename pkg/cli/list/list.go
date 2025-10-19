package list

import (
	"fmt"

	"github.com/clintharrison/go-kindle-pkg/pkg/cli/clicommon"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List available packages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			repo, err := clicommon.GetRepoFromArgs(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to initialize package repository")
			}
			packages, err := repo.FetchPackages(cmd.Context())
			if err != nil {
				return errors.Wrap(err, "failed to list packages")
			}

			// now we group by repo then package for nicer printing
			repos := map[string]map[string][]*repository.PackageArtifact{}
			for _, p := range packages {
				if repos[p.RepositoryID] == nil {
					repos[p.RepositoryID] = map[string][]*repository.PackageArtifact{}
				}
				repos[p.RepositoryID][p.ID] = append(repos[p.RepositoryID][p.ID], p)
			}

			for repoID, packages := range repos {
				fmt.Printf("\u001b[1mRepository: %s\u001b[0m\n", repoID)
				for p, as := range packages {
					// you probably shouldn't have an empty package ID, but anyway...
					if p == "" {
						p = "<no package ID>"
					}
					fmt.Printf("%s:\n", p)
					for _, a := range as {
						fmt.Printf("  %s\n", a.Version.String())
					}
				}
			}

			return nil
		},
	}
	return cmd
}
