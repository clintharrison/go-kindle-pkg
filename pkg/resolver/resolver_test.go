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
		ID:            ArtifactID(id),
		RepositoryID:  "",
		Version:       mkSV(major, minor, patch),
		Dependencies:  deps,
		SupportedArch: nil,
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
					ID:            a.ID,
					Version:       a.Version,
					RepositoryID:  "",
					SupportedArch: nil,
					Dependencies:  nil,
				})
			}
			resultList := []VersionedPackage{}
			for _, a := range result {
				resultList = append(resultList, VersionedPackage{
					ID:            a.ID,
					Version:       a.Version,
					RepositoryID:  "",
					SupportedArch: nil,
					Dependencies:  nil,
				})
			}
			require.ElementsMatch(t, expectedList, resultList, "Expected %v, got %v", expectedList, resultList)
		})
	}
}

func TestDiffInstallations_InstallsInDependencyOrder(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	tt := []struct {
		name         string
		universe     []*VersionedPackage
		current      map[ArtifactID][]*VersionedPackage
		constraints  []*Constraint
		expectedAdd  []string
		expectedRm   []string
		expectedFunc func(require.TestingT, []string, []string)
	}{
		{
			name: "simple linear dependencies",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
				mkPkgA("pkgB", 1, 0, 0, mkMinC("pkgC", 1, 0, 0)),
				mkPkgA("pkgC", 1, 0, 0, mkMinC("pkgD", 1, 0, 0)),
				mkPkgA("pkgD", 1, 0, 0, mkMinC("pkgE", 1, 0, 0)),
				mkPkgA("pkgE", 1, 0, 0),
			},
			constraints: []*Constraint{
				mkMinC("pkgA", 1, 0, 0),
			},
			expectedAdd: []string{
				"pkgE-1.0.0",
				"pkgD-1.0.0",
				"pkgC-1.0.0",
				"pkgB-1.0.0",
				"pkgA-1.0.0",
			},
		},

		{
			name: "branching dependencies",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
				mkPkgA("pkgB", 1, 0, 0, mkMinC("pkgC", 1, 0, 0), mkMinC("pkgD", 1, 0, 0)),
				mkPkgA("pkgC", 1, 0, 0, mkMinC("pkgD", 1, 0, 0)),
				mkPkgA("pkgD", 1, 0, 0),
			},

			constraints: []*Constraint{
				mkMinC("pkgD", 1, 0, 0),
				mkMinC("pkgA", 1, 0, 0),
			},

			expectedAdd: []string{
				"pkgD-1.0.0",
				"pkgC-1.0.0",
				"pkgB-1.0.0",
				"pkgA-1.0.0",
			},
		},

		{
			name: "with removal",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
				mkPkgA("pkgB", 1, 0, 0, mkMinC("pkgC", 1, 0, 0)),
				mkPkgA("pkgC", 1, 0, 0),
				mkPkgA("pkgD", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
			},

			constraints: []*Constraint{
				mkMinC("pkgA", 1, 0, 0),
			},
			current: map[ArtifactID][]*VersionedPackage{
				ArtifactID("pkgC"): {mkPkgA("pkgC", 0, 9, 9)},
			},

			expectedAdd: []string{
				"pkgC-1.0.0",
				"pkgB-1.0.0",
				"pkgA-1.0.0",
			},
			expectedRm: []string{
				"pkgC-0.9.9",
			},
		},

		{
			name: "with complex removal",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
				mkPkgA("pkgB", 1, 0, 0, mkMinC("pkgC", 1, 0, 0)),
				mkPkgA("pkgC", 1, 0, 0),
				mkPkgA("pkgD", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
			},

			constraints: []*Constraint{},
			current: map[ArtifactID][]*VersionedPackage{
				ArtifactID("pkgA"): {mkPkgA("pkgA", 1, 0, 0, mkMinC("pkgB", 1, 0, 0))},
				ArtifactID("pkgB"): {mkPkgA("pkgB", 1, 0, 0, mkMinC("pkgC", 1, 0, 0))},
				ArtifactID("pkgC"): {mkPkgA("pkgC", 1, 0, 0)},
				ArtifactID("pkgD"): {mkPkgA("pkgD", 1, 0, 0, mkMinC("pkgA", 1, 0, 0))},
			},

			expectedRm: []string{
				"pkgC-1.0.0",
				"pkgB-1.0.0",
				"pkgA-1.0.0",
				"pkgD-1.0.0",
			},
		},

		{
			name: "with loop",
			universe: []*VersionedPackage{
				mkPkgA("pkgA", 1, 0, 0, mkMinC("pkgB", 1, 0, 0)),
				mkPkgA("pkgB", 1, 0, 0, mkMinC("pkgC", 1, 0, 0), mkMinC("pkgD", 1, 0, 0)),
				mkPkgA("pkgC", 1, 0, 0, mkMinC("pkgD", 1, 0, 0)),
				mkPkgA("pkgD", 1, 0, 0, mkMinC("pkgC", 1, 0, 0)),
			},

			constraints: []*Constraint{
				mkMinC("pkgD", 1, 0, 0),
				mkMinC("pkgA", 1, 0, 0),
			},

			expectedFunc: func(t require.TestingT, add, _ []string) {
				require.Len(t, add, 4, "should be four installs")
				// pkgC and pkgD can be in either order due to the loop
				require.ElementsMatch(t, add[0:2], []string{"pkgC-1.0.0", "pkgD-1.0.0"},
					"first two installs should be pkgC and pkgD in any order")
				require.ElementsMatch(t, add[2:], []string{"pkgB-1.0.0", "pkgA-1.0.0"},
					"last two installs should be pkgB and pkgA in any order")
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := NewResolver(test.universe)
			result, err := r.Resolve(test.constraints)
			require.NoError(t, err)
			add, rm := DiffInstallations(test.current, result)
			require.NoError(t, err)

			var addIDs []string
			for _, a := range add {
				addIDs = append(addIDs, a.String())
			}
			if test.expectedAdd != nil {
				require.Equal(t, test.expectedAdd, addIDs)
			}

			var rmIDs []string
			for _, a := range rm {
				rmIDs = append(rmIDs, a.String())
			}
			if test.expectedRm != nil {
				require.Equal(t, test.expectedRm, rmIDs)
			}
			if test.expectedFunc != nil {
				test.expectedFunc(t, addIDs, rmIDs)
			}
		})
	}
}
