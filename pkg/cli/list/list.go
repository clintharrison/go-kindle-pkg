package list

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/clintharrison/go-kindle-pkg/pkg/cli/clicommon"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
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

			installedOnly, err := cmd.Flags().GetBool("installed")
			if err != nil {
				return errors.Wrap(err, "failed to get installed flag")
			}
			if installedOnly {
				packages, err := getInstalledPackages()
				if err != nil {
					return errors.Wrap(err, "failed to get installed packages")
				}
				for p, as := range packages {
					if p == "" {
						p = "<no package ID>"
					}
					fmt.Printf("%s:\n", p)
					for _, a := range as {
						fmt.Printf("  %s\n", a.Version.String())
					}
				}
			} else {
				repos, err := getAvailablePackages(cmd.Context(), repo)
				if err != nil {
					return errors.Wrap(err, "failed to get available packages")
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
			}

			return nil
		},
	}
	cmd.Flags().BoolP("installed", "i", false, "List installed packages only")
	return cmd
}

func getInstalledPackages() (map[string][]*repository.RepoPackage, error) {
	pkgs := make(map[string][]*repository.RepoPackage)
	baseDir := filepath.Join(version.BaseDir(), "pkgs")
	pkgsFS := os.DirFS(baseDir)
	fs.WalkDir(pkgsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, "/manifest.json") {
			slog.Debug("found installed package manifest", "path", path)
			data, err := fs.ReadFile(pkgsFS, path)
			if err != nil {
				return errors.AddStack(err)
			}
			var m manifest.Manifest
			err = json.Unmarshal(data, &m)
			if err != nil {
				return errors.AddStack(err)
			}
			ds := make([]repository.PackageDependency, 0, len(m.Dependencies))
			for _, d := range m.Dependencies {
				ds = append(ds, repository.PackageDependency{
					ID:           d.ID,
					RepositoryID: d.RepositoryID,
					Min:          d.Min,
					Max:          d.Max,
				})
			}
			pkg := &repository.RepoPackage{
				ID:           m.ID,
				Version:      m.Version,
				RepositoryID: "<installed>",
				Dependencies: ds,
			}
			pkgs[m.ID] = append(pkgs[m.ID], pkg)
		}
		return nil
	})
	return pkgs, nil
}

func getAvailablePackages(
	ctx context.Context, repo repository.Repository,
) (map[string]map[string][]*repository.RepoPackage, error) {
	packages, err := repo.FetchPackages(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list packages")
	}

	// now we group by repo then package for nicer printing
	repos := map[string]map[string][]*repository.RepoPackage{}
	for _, p := range packages {
		if repos[p.RepositoryID] == nil {
			repos[p.RepositoryID] = map[string][]*repository.RepoPackage{}
		}
		repos[p.RepositoryID][p.ID] = append(repos[p.RepositoryID][p.ID], p)
	}

	return repos, nil
}
