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
	fmt.Fprint(cmd.OutOrStdout(), "Using packages from repositories:\n") //nolint:errcheck
	for _, u := range repoURLs {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", u) //nolint:errcheck
	}

	repo, err := repository.NewFromURLs(repoURLs...)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), //nolint:errcheck
			"ERROR: Unable to create repositories:\n%v\n",
			err)
		return nil, errors.Wrap(err, "failed to create repository from URLs")
	}
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
	fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d package\n", len(packages)) //nolint:errcheck

	return resolver.NewResolverForRepositoryPackages(packages), nil
}
