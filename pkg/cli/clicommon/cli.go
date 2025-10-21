package clicommon

import (
	"fmt"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/resolver"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func GetRepoFromArgs(cmd *cobra.Command) (*repository.MultiRepository, error) {
	repoURLs, err := cmd.Flags().GetStringArray("repo")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repo URLs")
	}

	var rs []repository.Repository
	for _, url := range repoURLs {
		r, err := repository.NewHTTPRepository(url)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), //nolint:errcheck
				"ERROR: Unable to create repository for URL %s:\n%v\n",
				url, err)
			return nil, errors.Wrapf(err, "failed to create repository for URL %s", url)
		}
		rs = append(rs, r)
	}
	repo := repository.NewMultiRepository(rs...)
	return repo, nil
}

func GetInitializedResolver(cmd *cobra.Command) (*resolver.Resolver, error) {
	repo, err := GetRepoFromArgs(cmd)
	if err != nil {
		return nil, err
	}
	packages, err := repo.FetchPackages(cmd.Context())
	if err != nil {
		fmt.Fprintf( //nolint:errcheck
			cmd.OutOrStderr(),
			"ERROR: Unable to fetch packages from repositories:\n%v\n",
			err)
		return nil, errors.Wrap(err, "failed to fetch packages from repositories")
	}
	suffix := ""
	if len(packages) > 1 {
		suffix = "s"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d package%s\n", len(packages), suffix) //nolint:errcheck

	return resolver.NewResolverForRepositoryPackages(packages), nil
}
