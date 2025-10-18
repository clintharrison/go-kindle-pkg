package resolve

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
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

			// convert from one type to another, sigh...
			resolverPackages, err := resolverPackagesForRepository(packages)
			if err != nil {
				return errors.Wrap(err, "failed to convert packages to resolver artifacts")
			}

			r := resolver.NewResolver(resolverPackages)

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

var constraintRegexp = regexp.MustCompile(`^(?<package_id>[a-z-.]+)(?:[\s,]*(?:(?:==?\s*(?<eql>[\d.]+))|(?:>=\s*(?<min>[\d.]+))|(?:\<\s*(?<max>[\d.]+)))[\s,]*)*$`)

func parseVersion(vstr string) (*manifest.SemanticVersion, error) {
	sv := &manifest.SemanticVersion{}
	// handle 1, 1.0, 1.0.0
	// split on '.' and parse up to three components
	parts := strings.Split(vstr, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		v, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil, fmt.Errorf("invalid version component %q: %w", parts[i], err)
		}
		switch i {
		case 0:
			sv.Major = v
		case 1:
			sv.Minor = v
		case 2:
			sv.Patch = v
		}
	}
	return sv, nil
}

// parseConstraint handles a very basic spec for now:
//
//	package-id
//	package-id=version (or ==)
//	package-id>=version (must be >=)
//	package-id<version  (must only be <)
//	package-id>=1.0.0,<2.0.0 (combined constraints, order doesn't matter)
func parseConstraint(arg string) (*resolver.Constraint, error) {
	matches := constraintRegexp.FindStringSubmatch(arg)
	if matches == nil {
		return nil, fmt.Errorf("unable to parse constraint from arg %q", arg)
	}

	c := resolver.Constraint{}
	c.ID = resolver.ArtifactID(matches[constraintRegexp.SubexpIndex("package_id")])

	if eql := matches[constraintRegexp.SubexpIndex("eql")]; eql != "" {
		// eql will be the numeric portion from the regexp (e.g. "1.2.3")
		sv, err := parseVersion(eql)
		if err != nil {
			return nil, fmt.Errorf("unable to parse equality version from arg %q: %w", arg, err)
		}
		c.Min = sv
		c.Max = &manifest.SemanticVersion{
			Major: sv.Major,
			Minor: sv.Minor,
			Patch: sv.Patch + 1,
		}
		return &c, nil
	}

	if min := matches[constraintRegexp.SubexpIndex("min")]; min != "" {
		sv, err := parseVersion(min)
		if err != nil {
			return nil, fmt.Errorf("unable to parse minimum version from arg %q: %w", arg, err)
		}
		c.Min = sv
	}

	if max := matches[constraintRegexp.SubexpIndex("max")]; max != "" {
		sv, err := parseVersion(max)
		if err != nil {
			return nil, fmt.Errorf("unable to parse maximum version from arg %q: %w", arg, err)
		}
		c.Max = sv
	}

	return &c, nil
}

func constraintsFromArgs(args []string) ([]*resolver.Constraint, error) {
	var constraints []*resolver.Constraint
	for _, arg := range args {
		c, err := parseConstraint(arg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse constraint from arg %q", arg)
		}
		constraints = append(constraints, c)
	}
	return constraints, nil
}

// resolverPackagesForRepository converts a list of repository.PackageArtifact to resolver.Artifact
func resolverPackagesForRepository(packages []*repository.PackageArtifact) ([]*resolver.Artifact, error) {
	var res []*resolver.Artifact
	for _, pa := range packages {
		ds := make([]*resolver.Constraint, len(pa.Dependencies))
		for i, d := range pa.Dependencies {
			ds[i] = &resolver.Constraint{
				ID:  resolver.ArtifactID(d.ID),
				Min: d.Min,
				Max: d.Max,
			}
			if d.RepositoryID != nil {
				rid := resolver.RepositoryID(*d.RepositoryID)
				ds[i].RepositoryID = &rid
			}
		}
		ra := &resolver.Artifact{
			ID:           resolver.ArtifactID(pa.ID),
			RepositoryID: resolver.RepositoryID(pa.RepositoryID),
			Version:      pa.Version,
			Dependencies: ds,
		}
		res = append(res, ra)
	}
	return res, nil
}
