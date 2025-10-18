package resolver

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/pingcap/errors"
)

type (
	ArtifactID   string
	RepositoryID string
)

type Artifact struct {
	ID           ArtifactID
	RepositoryID RepositoryID
	// version is not optional!
	Version      manifest.SemanticVersion
	Dependencies []*Constraint
}

func (a *Artifact) String() string {
	return fmt.Sprintf("%s-%d.%d.%d", a.ID, a.Version.Major, a.Version.Minor, a.Version.Patch)
}

type Constraint struct {
	ID ArtifactID
	// inclusive lower bound
	Min *manifest.SemanticVersion
	// exclusive upper bound
	Max *manifest.SemanticVersion
	// optional repository to restrict search to
	RepositoryID *RepositoryID
}

// Allows checks whether an artifact matches this constraint.
// TODO: Move this to an interface method and out of the resolver package; Constraint can be generic perhaps?
func (c *Constraint) Allows(art *Artifact) bool {
	if c.Min != nil && art.Version.Compare(*c.Min) < 0 {
		return false
	}
	if c.Max != nil && art.Version.Compare(*c.Max) >= 0 {
		return false
	}
	if c.RepositoryID != nil && art.RepositoryID != *c.RepositoryID {
		return false
	}
	return true
}

func (c *Constraint) String() string {
	minStr := "none"
	maxStr := "none"
	if c.Min != nil {
		minStr = fmt.Sprintf("%d.%d.%d", c.Min.Major, c.Min.Minor, c.Min.Patch)
	}
	if c.Max != nil {
		maxStr = fmt.Sprintf("%d.%d.%d", c.Max.Major, c.Max.Minor, c.Max.Patch)
	}
	return fmt.Sprintf("%s [min=%s, max=%s]", c.ID, minStr, maxStr)
}

type Resolver struct {
	packages         map[ArtifactID][]*Artifact
	preferMaxVersion bool
}

func NewResolverForRepositoryPackages(packages []*repository.PackageArtifact) *Resolver {
	var res []*Artifact
	for _, pa := range packages {
		ds := make([]*Constraint, len(pa.Dependencies))
		for i, d := range pa.Dependencies {
			var rid *RepositoryID
			if d.RepositoryID != nil {
				rid = (*RepositoryID)(d.RepositoryID)
			}
			ds[i] = &Constraint{
				ID:           ArtifactID(d.ID),
				Min:          d.Min,
				Max:          d.Max,
				RepositoryID: rid,
			}
		}
		ra := &Artifact{
			ID:           ArtifactID(pa.ID),
			RepositoryID: RepositoryID(pa.RepositoryID),
			Version:      pa.Version,
			Dependencies: ds,
		}
		res = append(res, ra)
	}
	return NewResolver(res)
}

func NewResolver(universe []*Artifact) *Resolver {
	r := &Resolver{
		packages: make(map[ArtifactID][]*Artifact),
		// candidates are sorted descending by version
		preferMaxVersion: true,
	}
	for _, a := range universe {
		r.packages[a.ID] = append(r.packages[a.ID], a)
	}
	slog.Debug("resolver packages", "count", len(universe))
	for _, p := range r.packages {
		for _, v := range p {
			slog.Debug("package version", "package", v.ID, "version", v, "dependencies", v.Dependencies)
		}
	}
	return r
}

func (r *Resolver) Resolve(constraints []*Constraint) (map[ArtifactID]*Artifact, error) {
	// initial empty state
	resolved := map[ArtifactID]*Artifact{}
	res, success := r.resolveRecursive(constraints, resolved)
	if !success {
		return nil, errors.Errorf("unable to resolve desired packages")
	}
	return res, nil
}

// resolvedRecursive takes the remaining unresolved constraints and the current resolved map,
// and attempts to resolve all constraints recursively, returning the final resolved map or an error.
func (r *Resolver) resolveRecursive(
	constraints []*Constraint, resolved map[ArtifactID]*Artifact,
) (map[ArtifactID]*Artifact, bool) {
	slog.Debug("resolveRecursive called", "constraints", constraints, "resolved", resolved)
	if len(constraints) == 0 {
		// no more constraints :)
		return resolved, true
	}

	// pop the next constraint to work on
	constraint, constraintsRemaining := constraints[0], constraints[1:]
	cid := constraint.ID

	if _, ok := r.packages[cid]; !ok {
		slog.Error("Unknown package requested", "package", cid, "constraint", constraint)
		return nil, false
	}

	// we have a dependency for a package that's already resolved, and the new constraint is compatible
	if currVer, ok := resolved[cid]; ok {
		if constraint.Allows(currVer) {
			// "drop" this constraint and continue resolving the rest
			return r.resolveRecursive(constraintsRemaining, resolved)
		}
		// conflict! we'll need to backtrack
		return nil, false
	}

	// candidate list is ordered descending by version (by default)
	// TODO: consider repository order -- which must always be descending priority?
	candidates := make([]*Artifact, len(r.packages[cid]))
	copy(candidates, r.packages[cid])
	slices.SortFunc(candidates, func(a, b *Artifact) int {
		if r.preferMaxVersion {
			return b.Version.Compare(a.Version)
		}
		return a.Version.Compare(b.Version)
	})

	for _, candidate := range candidates {
		if !constraint.Allows(candidate) {
			slog.Debug("skipping candidate that does not satisfy constraint", "constraint", constraint, "candidate", candidate)
			continue
		}

		// tentatively select this candidate: this may be backtracked
		resolved[cid] = candidate
		slog.Debug("tentatively selected candidate", "candidate", candidate, "dependencies", candidate.Dependencies)

		newConstraints := make([]*Constraint, 0, len(constraintsRemaining)+len(candidate.Dependencies))
		newConstraints = append(newConstraints, constraintsRemaining...)
		newConstraints = append(newConstraints, candidate.Dependencies...)
		res, success := r.resolveRecursive(newConstraints, resolved)
		if success {
			return res, true
		}

		// backtrack selection of this candidate
		delete(resolved, cid)
		slog.Debug("backtracking on candidate", "candidate", candidate)
	}

	// no candidates were successful
	return nil, false
}
