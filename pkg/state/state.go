package state

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/pingcap/errors"
)

func GetInstalledPackages() (map[string][]*repository.RepoPackage, error) {
	// TODO: represent "external" packages (e.g. koreader from a legacy install)
	pkgs := make(map[string][]*repository.RepoPackage)
	baseDir := filepath.Join(version.BaseDir(), "pkgs")
	pkgsFS := os.DirFS(baseDir)
	err := fs.WalkDir(pkgsFS, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return errors.AddStack(err)
		}
		if strings.HasSuffix(path, "/manifest.json") {
			slog.Debug("found installed package manifest", "path", path)
			data, err := fs.ReadFile(pkgsFS, path)
			if err != nil {
				return errors.AddStack(err)
			}
			var m manifest.Manifest
			err = json.Unmarshal(data, &m)
			if err != nil {
				return errors.AddStack(err)
			}
			ds := make([]repository.PackageDependency, 0, len(m.Dependencies))
			for _, d := range m.Dependencies {
				ds = append(ds, repository.PackageDependency{
					ID:           d.ID,
					RepositoryID: d.RepositoryID,
					Min:          d.Min,
					Max:          d.Max,
				})
			}
			pkg := &repository.RepoPackage{
				ID:            m.ID,
				Version:       m.Version,
				RepositoryID:  "<installed>",
				SupportedArch: m.SupportedArch,
				Dependencies:  ds,
			}
			pkgs[m.ID] = append(pkgs[m.ID], pkg)
		}
		return nil
	})
	if err != nil {
		return nil, errors.AddStack(err)
	}
	return pkgs, nil
}
