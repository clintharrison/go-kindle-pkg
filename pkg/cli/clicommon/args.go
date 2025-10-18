package clicommon

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/resolver"
	"github.com/pingcap/errors"
)

var constraintRegexp = regexp.MustCompile(
	`^(?<package_id>[a-z-.]+)` +
		`(?:[\s,]*(?:` +
		// = or ==1.2.3
		`(?:==?\s*(?<eql>[\d.]+))` +
		// >=1.2.3
		`|(?:>=\s*(?<min>[\d.]+))` +
		// <1.2.3
		`|(?:\<\s*(?<max>[\d.]+))` +
		// comma and spaces are allowed between constraints
		`)[\s,]*)*$`)

// ParseConstraint handles a very basic spec for now:
//
//	package-id
//	package-id=version (or ==)
//	package-id>=version (must be >=)
//	package-id<version  (must only be <)
//	package-id>=1.0.0,<2.0.0 (combined constraints, order doesn't matter)
func ParseConstraint(arg string) (*resolver.Constraint, error) {
	matches := constraintRegexp.FindStringSubmatch(arg)
	if matches == nil {
		return nil, fmt.Errorf("unable to parse constraint from arg %q", arg)
	}

	c := resolver.Constraint{} //nolint:exhaustruct
	c.ID = resolver.ArtifactID(matches[constraintRegexp.SubexpIndex("package_id")])

	if eql := matches[constraintRegexp.SubexpIndex("eql")]; eql != "" {
		// eql will be the numeric portion from the regexp (e.g. "1.2.3")
		sv, err := ParseVersion(eql)
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

	if match := matches[constraintRegexp.SubexpIndex("min")]; match != "" {
		sv, err := ParseVersion(match)
		if err != nil {
			return nil, fmt.Errorf("unable to parse minimum version from arg %q: %w", arg, err)
		}
		c.Min = sv
	}

	if match := matches[constraintRegexp.SubexpIndex("max")]; match != "" {
		sv, err := ParseVersion(match)
		if err != nil {
			return nil, fmt.Errorf("unable to parse maximum version from arg %q: %w", arg, err)
		}
		c.Max = sv
	}

	return &c, nil
}

func ParseVersion(vstr string) (*manifest.SemanticVersion, error) {
	sv := &manifest.SemanticVersion{} //nolint:exhaustruct
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

func ConstraintsFromArgs(args []string) ([]*resolver.Constraint, error) {
	var constraints []*resolver.Constraint
	for _, arg := range args {
		c, err := ParseConstraint(arg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse constraint from arg %q", arg)
		}
		constraints = append(constraints, c)
	}
	return constraints, nil
}
