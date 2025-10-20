package resolver

import (
	"testing"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/stretchr/testify/require"
)

func mkSV(major, minor, patch int) manifest.SemanticVersion {
	return manifest.SemanticVersion{Major: major, Minor: minor, Patch: patch}
}

func mkC(id string) *Constraint {
	return &Constraint{ID: ArtifactID(id), RepositoryID: nil, Min: nil, Max: nil}
}

func mkMinC(id string, major, minor, patch int) *Constraint {
	sv := mkSV(major, minor, patch)
	return &Constraint{ID: ArtifactID(id), RepositoryID: nil, Min: &sv, Max: nil}
}

func mkMaxC(id string, major, minor, patch int) *Constraint { //nolint:unparam
	sv := mkSV(major, minor, patch)
	return &Constraint{ID: ArtifactID(id), RepositoryID: nil, Min: nil, Max: &sv}
}

func mkMinMaxC(
	id string, minMajor, minMinor, minPatch int, maxMajor, maxMinor, maxPatch int, //nolint:unparam
) *Constraint {
	minSV := mkSV(minMajor, minMinor, minPatch)
	maxSV := mkSV(maxMajor, maxMinor, maxPatch)
	return &Constraint{ID: ArtifactID(id), RepositoryID: nil, Min: &minSV, Max: &maxSV}
}

func mkPkgA(id string, major, minor, patch int, deps ...*Constraint) *VersionedPackage {
	return &VersionedPackage{
		ID:           ArtifactID(id),
		RepositoryID: "",
		Version:      mkSV(major, minor, patch),
		Dependencies: deps,
	}
}

func TestResolver(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// input
		universe        []*VersionedPackage
		desiredPackages []*Constraint
		// output
		expected    []*VersionedPackage
		expectError bool
	}{
		{
			name: "matches highest possible version",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0),
				mkPkgA("pkgA", 1, 1, 0),
			},
			desiredPackages: []*Constraint{
				mkMinC("pkgA", 1, 0, 0),
			},
			expected: []*VersionedPackage{
				mkPkgA("pkgA", 1, 1, 0),
			},
			expectError: false,
		},

		{
			name: "matches only possible version",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0),
				mkPkgA("pkgA", 1, 1, 0),
				mkPkgA("pkgB", 1, 1, 0),
				mkPkgA("pkgC", 2, 0, 1),
			},
			desiredPackages: []*Constraint{
				mkMinC("pkgB", 1, 1, 0),
			},
			expected: []*VersionedPackage{
				mkPkgA("pkgB", 1, 1, 0),
			},
			expectError: false,
		},

		{
			name: "matches version with max constraint",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0),
				mkPkgA("pkgA", 1, 1, 0),
			},
			desiredPackages: []*Constraint{
				mkMinC("pkgA", 0, 0, 0),
				mkMaxC("pkgA", 1, 1, 0),
			},
			expected: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0),
			},
			expectError: false,
		},

		{
			name: "includes the latest dependency",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("libdep", 2, 0, 0)),
				mkPkgA("libdep", 2, 0, 0),
				mkPkgA("libdep", 2, 0, 10),
			},
			desiredPackages: []*Constraint{
				// 0.0.0 is a request for the latest version
				mkMinC("pkgA", 0, 0, 0),
			},
			expected: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("libdep", 2, 0, 0)),
				mkPkgA("libdep", 2, 0, 10),
			},
			expectError: false,
		},

		{
			name: "reproduces sample repo JSON",
			universe: []*VersionedPackage{
				mkPkgA(
					"com.kindlemodding.examplepackage",
					1, 2, 3,
					mkMinMaxC("io.github.niluje.fbink", 0, 6, 10, 0, 7, 0),
					mkC("org.lua"),
					mkMinC("testmin", 1, 0, 0),
					mkMaxC("testmax", 1, 0, 0),
				),
				mkPkgA(
					"io.github.niluje.fbink",
					0, 6, 9,
					mkMinC("testmin", 1, 9, 0),
					mkMaxC("testmax", 1, 0, 0),
				),
				mkPkgA(
					"io.github.niluje.fbink",
					0, 6, 10,
					mkMinC("testmin", 1, 0, 0),
					mkMaxC("testmax", 1, 0, 0),
				),
				mkPkgA(
					"io.github.niluje.fbink",
					0, 6, 11,
					mkMinC("testmin", 1, 0, 0),
					mkMaxC("testmax", 1, 0, 0),
				),
				mkPkgA(
					"org.lua",
					9, 2, 3,
					mkMinMaxC("testmin", 1, 0, 1, 1, 9, 0),
				),
				mkPkgA(
					"org.lua",
					1, 2, 5,
					mkMinMaxC("testmin", 1, 0, 1, 1, 9, 0),
				),
				mkPkgA(
					"org.lua",
					4, 5, 6,
					mkMinMaxC("testmin", 1, 9, 0, 2, 0, 0),
				),
				mkPkgA("testmin", 0, 1, 2),
				mkPkgA("testmin", 0, 2, 3),
				mkPkgA("testmin", 0, 99, 99),
				mkPkgA("testmin", 1, 0, 0),
				mkPkgA("testmin", 1, 1, 1),
				mkPkgA("testmin", 1, 999, 999),

				mkPkgA("testmax", 0, 1, 2),
				mkPkgA("testmax", 0, 2, 3),
				mkPkgA("testmax", 0, 99, 99),
				mkPkgA("testmax", 1, 0, 0),
				mkPkgA("testmax", 1, 1, 1),
				mkPkgA("testmax", 1, 999, 999),
			},
			desiredPackages: []*Constraint{
				mkMinC("com.kindlemodding.examplepackage", 1, 2, 3),
			},
			expected: []*VersionedPackage{
				mkPkgA("com.kindlemodding.examplepackage", 1, 2, 3),
				mkPkgA("testmax", 0, 99, 99),
				mkPkgA("io.github.niluje.fbink", 0, 6, 11),
				mkPkgA("org.lua", 9, 2, 3),
				mkPkgA("testmin", 1, 1, 1),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewResolver(tt.universe)
			result, err := r.Resolve(tt.desiredPackages)
			if (err != nil) != tt.expectError {
				t.Errorf("Resolve() error = %v, expectError %v", err, tt.expectError)
				return
			}
			// for the sake of these tests, we don't care about the dependencies when we're checking
			// the result set, just the IDs and Versions.
			// (this makes writing the expected results easier)
			expectedList := []VersionedPackage{}
			for _, a := range tt.expected {
				expectedList = append(expectedList, VersionedPackage{
					ID:           a.ID,
					Version:      a.Version,
					RepositoryID: "",
					Dependencies: nil,
				})
			}
			resultList := []VersionedPackage{}
			for _, a := range result {
				resultList = append(resultList, VersionedPackage{
					ID:           a.ID,
					Version:      a.Version,
					RepositoryID: "",
					Dependencies: nil,
				})
			}
			require.ElementsMatch(t, expectedList, resultList, "Expected %v, got %v", expectedList, resultList)
		})
	}
}
