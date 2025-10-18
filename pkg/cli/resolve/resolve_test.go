package resolve

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/clintharrison/go-kindle-pkg/pkg/cli/clicommon"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	repositorytestdata "github.com/clintharrison/go-kindle-pkg/pkg/repository/testdata"
	"github.com/clintharrison/go-kindle-pkg/pkg/resolver"
	"github.com/stretchr/testify/require"
)

// parseConstraint handles a very basic spec for now:
//
//	package-id
//	package-id=version (or ==)
//	package-id>=version (must be >=)
//	package-id<version  (must only be <)
//	package-id>=1.0.0,<2.0.0 (combined constraints, order doesn't matter)
//
// These tests exercise the current behaviour of parseConstraint. Some
// parsing behaviours (particularly version parsing) are known to be
// imperfect; the tests assert the present outputs so they remain
// stable across refactors unless the implementation is intentionally
// changed.

func mkSV(major, minor, patch int) manifest.SemanticVersion {
	return manifest.SemanticVersion{Major: major, Minor: minor, Patch: patch}
}

func mkPtr[T any](t T) *T { return &t }

//nolint:exhaustruct
func TestParseConstraint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		arg         string
		expectError bool
		wantID      resolver.ArtifactID
		wantMin     *manifest.SemanticVersion
		wantMax     *manifest.SemanticVersion
	}{
		{
			name:        "bare package id",
			arg:         "pkg",
			expectError: false,
			wantID:      resolver.ArtifactID("pkg"),
			wantMin:     nil,
			wantMax:     nil,
		},
		{
			name:        "equality single =",
			arg:         "pkg=1.2.3",
			expectError: false,
			wantID:      resolver.ArtifactID("pkg"),
			wantMin:     mkPtr(mkSV(1, 2, 3)),
			wantMax:     mkPtr(mkSV(1, 2, 4)),
		},
		{
			name:        "equality double ==",
			arg:         "pkg==2.0.0",
			expectError: false,
			wantID:      resolver.ArtifactID("pkg"),
			wantMin:     mkPtr(mkSV(2, 0, 0)),
			wantMax:     mkPtr(mkSV(2, 0, 1)),
		},
		{
			name:        ">= min constraint",
			arg:         "pkg>=1.0.0",
			expectError: false,
			wantID:      resolver.ArtifactID("pkg"),
			wantMin:     mkPtr(mkSV(1, 0, 0)),
			wantMax:     nil,
		},
		{
			name:        "< max constraint",
			arg:         "pkg<2.0.0",
			expectError: false,
			wantID:      resolver.ArtifactID("pkg"),
			wantMin:     nil,
			wantMax:     mkPtr(mkSV(2, 0, 0)),
		},
		{
			name:        "combined constraints",
			arg:         "pkg>=1.0.0,<2.0.0",
			expectError: false,
			wantID:      resolver.ArtifactID("pkg"),
			wantMin:     mkPtr(mkSV(1, 0, 0)),
			wantMax:     mkPtr(mkSV(2, 0, 0)),
		},
		{
			name:        "malformed input",
			arg:         "not-a-package!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := clicommon.ParseConstraint(tt.arg)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
			require.Equal(t, tt.wantID, c.ID)
			if tt.wantMin == nil {
				require.Nil(t, c.Min)
			} else {
				require.Equal(t, *tt.wantMin, *c.Min)
			}
			if tt.wantMax == nil {
				require.Nil(t, c.Max)
			} else {
				require.Equal(t, *tt.wantMax, *c.Max)
			}
		})
	}
}

func TestResolveE2E_File(t *testing.T) {
	t.Parallel()

	tmpdir := t.TempDir()
	reposJSONPath := tmpdir + "/repo.json"
	err := os.WriteFile(reposJSONPath, repositorytestdata.RepositoryJSON, 0o644) //nolint:gosec
	require.NoError(t, err)

	cmd := NewCommand()
	cmd.SetArgs([]string{
		"--repo", "file://" + reposJSONPath,
		"com.kindlemodding.examplepackage>=1.0.0",
	})

	buf := bytes.Buffer{}
	cmd.SetOut(&buf)
	err = cmd.Execute()
	require.NoError(t, err)

	require.Contains(t, buf.String(), "Resolved packages:")
	require.Contains(t, buf.String(), "  - testmax-0.99.99")
	require.Contains(t, buf.String(), "  - com.kindlemodding.examplepackage-1.2.3")
	require.Contains(t, buf.String(), "  - io.github.niluje.fbink-0.6.99")
	require.Contains(t, buf.String(), "  - org.lua-9.2.3")
	require.Contains(t, buf.String(), "  - testmin-1.1.1")
}

func TestResolveE2E_HTTP(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repository.json" {
			t.Errorf("Expected to request \"/repository.json\", got: %q", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected \"Accept: application/json\" header, got: %q", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write(repositorytestdata.RepositoryJSON) //nolint:errcheck
	}))
	defer server.Close()

	repoURL := server.URL + "/repository.json"

	cmd := NewCommand()
	cmd.SetArgs([]string{
		"--repo", repoURL,
		"com.kindlemodding.examplepackage>=1.0.0",
	})

	buf := bytes.Buffer{}
	cmd.SetOut(&buf)
	err := cmd.Execute()
	require.NoError(t, err)

	require.Contains(t, buf.String(), "Resolved packages:")
	require.Contains(t, buf.String(), "  - testmax-0.99.99")
	require.Contains(t, buf.String(), "  - com.kindlemodding.examplepackage-1.2.3")
	require.Contains(t, buf.String(), "  - io.github.niluje.fbink-0.6.99")
	require.Contains(t, buf.String(), "  - org.lua-9.2.3")
	require.Contains(t, buf.String(), "  - testmin-1.1.1")
}
