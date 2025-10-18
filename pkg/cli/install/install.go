package install

import (
	"fmt"
	"os"
	"strings"

	"github.com/clintharrison/go-kindle-pkg/pkg/cli/clicommon"
	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/resolver"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [flags] example.kpkg",
		Short: "Extract and install a .kpkg file",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := clicommon.GetInitializedResolver(cmd)
			if err != nil {
				return errors.Wrap(err, "failed to initialize resolver")
			}

			fileArgs, rest, err := findFileArgs(args)
			if err != nil {
				return errors.Wrap(err, "failed to parse file arguments")
			}

			// read metadata from .kpkg files to generate constraints and artifacts
			// used for resolution
			fileConstraints, fileArtifacts, err := processKPKGArgs(fileArgs)
			if err != nil {
				return err
			}

			// parse the human-friendly-ish constraints that remain on the command line
			constraints, err := clicommon.ConstraintsFromArgs(rest)
			if err != nil {
				return errors.Wrap(err, "failed to parse package constraints from args")
			}

			constraints = append(fileConstraints, constraints...)

			result, err := r.Resolve(constraints, resolver.WithArtifacts(fileArtifacts))
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "ERROR: Unable to resolve packages:\n%v\n", err) //nolint:errcheck
				return errors.Wrap(err, "failed to resolve packages")
			}

			cmd.OutOrStdout().Write([]byte("Resolved packages:\n")) //nolint:errcheck
			for _, art := range result {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", art) //nolint:errcheck
			}
			return errors.Errorf("not implemented")
		},
	}

	return cmd
}

func processKPKGArgs(fileArgs []string) ([]*resolver.Constraint, []*resolver.Artifact, error) {
	var manifests []*manifest.Manifest
	for _, f := range fileArgs {
		kpkg, err := kpkg.Open(f)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "kpkg.OpenKPKGFile(%q)", f)
		}
		defer kpkg.Close()
		pkgManifest := kpkg.Manifest
		if pkgManifest == nil {
			return nil, nil, fmt.Errorf("kpkg %q has no manifest", f)
		}
		manifests = append(manifests, pkgManifest)
	}

	constraints, err := constraintsFromKPKGFiles(manifests)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate constraints from .kpkg files")
	}

	var artifacts []*resolver.Artifact
	for _, pkgManifest := range manifests {
		var deps []*resolver.Constraint
		// TODO: is this right? Should dependencies be an array (not a map) in the manifest,
		// mirroring the shape of repository manifests?
		for depID, dep := range pkgManifest.Dependencies {
			deps = append(deps, &resolver.Constraint{
				ID:           resolver.ArtifactID(depID),
				Min:          dep.Min,
				Max:          dep.Max,
				RepositoryID: (*resolver.RepositoryID)(dep.RepositoryID),
			})
		}
		art := &resolver.Artifact{
			ID:           resolver.ArtifactID(pkgManifest.ID),
			RepositoryID: "$kpkgfile",
			Version:      pkgManifest.Version,
			Dependencies: deps,
		}
		artifacts = append(artifacts, art)
	}

	return constraints, artifacts, nil
}

// findFileArgs separates .kpkg file arguments from version constraint (foo=1.2.3) arguments.
func findFileArgs(args []string) (fileArgs []string, rest []string, err error) {
	for _, arg := range args {
		fi, _ := os.Stat(arg)
		exists := fi != nil && fi.Mode().IsRegular()

		if strings.HasSuffix(arg, ".kpkg") || exists {
			// if you try to pass a .kpkg file, it's an error to not exist (in other words,
			// we are explicitly skipping parsing it as a package name)
			if !exists {
				return nil, nil, fmt.Errorf("file %q does not exist", arg)
			}
			fileArgs = append(fileArgs, arg)
		} else {
			rest = append(rest, arg)
		}
	}
	return fileArgs, rest, nil
}

// constraintsFromKPKGFiles generates constraints from the given .kpkg files,
// including dependencies specified in their manifests.
func constraintsFromKPKGFiles(manifests []*manifest.Manifest) ([]*resolver.Constraint, error) {
	var constraints []*resolver.Constraint
	for _, pkgManifest := range manifests {
		cs, err := func() ([]*resolver.Constraint, error) {
			maxC := manifest.SemanticVersion{
				Major: pkgManifest.Version.Major,
				Minor: pkgManifest.Version.Minor,
				Patch: pkgManifest.Version.Patch + 1,
			}
			constraint := &resolver.Constraint{
				ID:  resolver.ArtifactID(pkgManifest.ID),
				Min: &pkgManifest.Version,
				Max: &maxC,
			}

			cs := []*resolver.Constraint{constraint}
			for depID, dep := range pkgManifest.Dependencies {
				cs = append(cs, &resolver.Constraint{
					ID:  resolver.ArtifactID(depID),
					Min: dep.Min,
					Max: dep.Max,
				})
			}
			return cs, nil
		}()
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, cs...)
	}
	return constraints, nil
}
