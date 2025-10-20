package repository

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/pingcap/errors"
	"github.com/stretchr/testify/require"
)

func createDummyKPKGFile(t *testing.T, path string, patch int) error {
	t.Helper()
	pkgDir := t.TempDir()
	kpkgMeta := &manifest.Manifest{
		Version:       manifest.SemanticVersion{Major: 1, Minor: 0, Patch: patch},
		ID:            "dummy-package",
		Name:          "Dummy Package",
		Author:        "",
		Description:   "",
		SupportedArch: nil,
		Dependencies:  nil,
	}
	manifestPath := pkgDir + "/manifest.json"
	manifestJSON, err := json.Marshal(kpkgMeta)
	require.NoError(t, err)
	err = os.WriteFile(manifestPath, manifestJSON, 0o644) //nolint:gosec
	require.NoError(t, err)

	err = kpkg.Build(t.Context(), pkgDir, path)
	if err != nil {
		return errors.AddStack(err)
	}
	return nil
}

func TestMultiRepository(t *testing.T) {
	t.Parallel()

	pkgA := t.TempDir() + "/packageA.kpkg"
	pkgB := t.TempDir() + "/packageB.kpkg"
	pkgC := t.TempDir() + "/packageC.kpkg"

	for i, p := range []string{pkgA, pkgB, pkgC} {
		err := createDummyKPKGFile(t, p, i)
		require.NoError(t, err)
	}

	repo := NewMultiRepository(
		NewLocalFileRepository(pkgA),
		NewLocalFileRepository(pkgB),
	)

	repo.AddRepository(
		NewLocalFileRepository(pkgC),
	)

	ctx := t.Context()
	pkgs, err := repo.FetchPackages(ctx)
	require.NoError(t, err)

	t.Logf("MultiRepository String(): %s", repo.String())
	require.Len(t, pkgs, 3, "expected one package for each .kpkg file in MultiRepository")
	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.ID+"-"+pkg.Version.String())
	}
	require.ElementsMatch(t, names, []string{
		"dummy-package-1.0.0",
		"dummy-package-1.0.1",
		"dummy-package-1.0.2",
	})
}
