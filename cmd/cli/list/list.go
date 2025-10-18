package list

import (
	"fmt"
	"os"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	repositorytestdata "github.com/clintharrison/go-kindle-pkg/pkg/repository/testdata"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List available packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURLs, err := cmd.Flags().GetStringArray("repo")
			if err != nil {
				// HACK: read the repo URLs from a file instead?
				f, err := os.CreateTemp("", "gokp-repo-list-*.json")
				if err != nil {
					return errors.Wrap(err, "failed to create temp file")
				}
				defer os.Remove(f.Name())
				f.Write(repositorytestdata.RepositoryJSON)
				repoURLs = []string{fmt.Sprintf("file://%s", f.Name())}
				// END HACK
				// return errors.Wrap(err, "failed to get repo URLs")
			}

			r, err := repository.NewFromURLs(repoURLs...)
			if err != nil {
				return errors.Wrap(err, "failed to create repository from URLs")
			}

			// network call, eek!
			ps, err := r.FetchPackages()
			if err != nil {
				return errors.Wrap(err, "failed to list packages")
			}

			// now we group by repo then package for nicer printing
			repos := map[string]map[string][]*repository.PackageArtifact{}
			for _, p := range ps {
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
	cmd.Flags().StringArrayP("repo", "r", []string{}, "Repository URL(s) to use (can be specified multiple times)")

	return cmd
}
