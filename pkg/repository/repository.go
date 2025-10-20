package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/clintharrison/go-kindle-pkg/pkg/kpkg"
	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/pingcap/errors"
)

// RepoPackage is a specific version of a package from a repository.
type RepoPackage struct {
	ID            string
	RepositoryID  string
	Version       manifest.SemanticVersion
	SupportedArch []string
	Dependencies  []PackageDependency
}

func NewRepoPackage(
	packageID string, repoID string, art *manifest.Artifact,
) *RepoPackage {
	pa := RepoPackage{
		ID:            packageID,
		RepositoryID:  repoID,
		Version:       art.Version,
		SupportedArch: art.SupportedArch,
		Dependencies:  nil,
	}

	deps := []PackageDependency{}
	for _, d := range art.Dependencies {
		deps = append(deps, PackageDependency{
			ID:           d.ID,
			Min:          d.Min,
			Max:          d.Max,
			RepositoryID: d.RepositoryID,
		})
	}
	pa.Dependencies = deps
	return &pa
}

func (rp *RepoPackage) String() string {
	return fmt.Sprintf("%s-%s (repo: %s)", rp.ID, rp.Version.String(), rp.RepositoryID)
}

// PackageDependency represents a constraint on a dependent package.
type PackageDependency struct {
	ID string
	// Optional repository: this should be used rarely!
	RepositoryID *string
	// Optional inclusive minimum version
	Min *manifest.SemanticVersion
	// Optional exclusive maximum version
	Max *manifest.SemanticVersion
}

type Repository interface {
	fmt.Stringer
	ID() string
	FetchPackages(ctx context.Context) ([]*RepoPackage, error)
	DownloadPackage(ctx context.Context, repoPackage *RepoPackage, destFile string, dryRun bool) error
}

const localFileRepoID = "$kpkgfile"

type LocalFileRepository struct {
	paths          []string
	pkgs           []*RepoPackage
	pathForPackage map[string]string
}

func NewLocalFileRepository(paths ...string) *LocalFileRepository {
	return &LocalFileRepository{
		paths:          paths,
		pathForPackage: make(map[string]string, len(paths)),
	}
}

func (r *LocalFileRepository) String() string {
	return fmt.Sprintf("LocalFileRepository(%v)", r.paths)
}

func (r *LocalFileRepository) ID() string {
	return localFileRepoID
}

func (r *LocalFileRepository) DownloadPackage(
	ctx context.Context, pkg *RepoPackage, destPath string, dryRun bool,
) error {
	slog.Debug("LocalFileRepository.DownloadPackage()",
		"package", pkg.ID, "version", pkg.Version.String(), "repo_id", pkg.RepositoryID, "pkg", pkg, "self", r)
	srcPath, ok := r.pathForPackage[pkg.ID]
	if !ok {
		return fmt.Errorf("package %s not found in local file repository", pkg.ID)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrapf(err, "os.Open(%q)", srcPath)
	}
	defer src.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return errors.Wrapf(err, "os.Create(%q)", destPath)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, src)
	if err != nil {
		return errors.Wrapf(err, "io.Copy() to %q", destPath)
	}
	return nil
}

func (r *LocalFileRepository) FetchPackages(ctx context.Context) ([]*RepoPackage, error) {
	for _, p := range r.paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, errors.Wrapf(err, "os.Stat(%q)", p)
		}
		if fi.IsDir() {
			// TODO: scan directory for .kpkg files?
			continue
		}
		k, err := kpkg.Open(p)
		if err != nil {
			return nil, errors.Wrapf(err, "kpkg.Open(%q)", p)
		}
		defer k.Close() //nolint:errcheck
		if k.Manifest == nil {
			return nil, errors.Errorf("kpkg file %q does not have a manifest", p)
		}
		deps := []manifest.Dependency{}
		for dID, d := range k.Manifest.Dependencies {
			d := manifest.Dependency{
				ID:           dID,
				Min:          d.Min,
				Max:          d.Max,
				RepositoryID: d.RepositoryID,
			}
			deps = append(deps, d)
		}

		artifact := &manifest.Artifact{
			URL: p,
			Version: manifest.SemanticVersion{
				Major: k.Manifest.Version.Major,
				Minor: k.Manifest.Version.Minor,
				Patch: k.Manifest.Version.Patch,
			},
			Dependencies: deps,
		}
		r.pkgs = append(r.pkgs, NewRepoPackage(k.Manifest.ID, localFileRepoID, artifact))
		r.pathForPackage[k.Manifest.ID] = p
	}

	return r.pkgs, nil
}

type HTTPRepository struct {
	url        *url.URL
	pas        []*RepoPackage
	repoConfig *manifest.RepositoryConfig
}

func NewHTTPRepository(rawurl string) (*HTTPRepository, error) {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawurl, err)
	}
	switch parsed.Scheme {
	case "http", "https", "file":
		return &HTTPRepository{url: parsed, pas: nil}, nil
	default:
		return nil, fmt.Errorf("invalid URL scheme %q in repo %q", parsed.Scheme, rawurl)
	}
}

func (r *HTTPRepository) String() string {
	return fmt.Sprintf("HTTPRepository(%v)", r.url)
}

func (mr *HTTPRepository) ID() string {
	return mr.repoConfig.ID
}

func (mr *HTTPRepository) FetchPackages(ctx context.Context) ([]*RepoPackage, error) {
	mr.pas = []*RepoPackage{}
	var repoConfig manifest.RepositoryConfig
	err := readJSONFromURL(ctx, mr.url, &repoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository from %q: %w", mr.url.String(), err)
	}
	mr.repoConfig = &repoConfig

	for id, pkg := range repoConfig.Packages {
		for _, art := range pkg.Artifacts {
			mr.pas = append(mr.pas, NewRepoPackage(id, repoConfig.ID, &art))
		}
	}
	return mr.pas, nil
}

func (mr *HTTPRepository) DownloadPackage(
	ctx context.Context, pkg *RepoPackage, destPath string, dryRun bool,
) error {
	slog.Debug("HTTPRepository.DownloadPackage()", "package", pkg.ID, "version", pkg.Version.String(),
		"repo_id", pkg.RepositoryID, "repo_config_id", mr.repoConfig.ID)
	if pkg.RepositoryID != mr.repoConfig.ID {
		return fmt.Errorf("package %s does not belong to repository %s",
			pkg.ID, mr.repoConfig.ID)
	}

	art := mr.findArtifact(pkg.ID, pkg.Version)
	slog.Debug("HTTPRepository.DownloadPackage()",
		"package", pkg.ID, "version", pkg.Version.String(), "artifact", art)

	if dryRun {
		fmt.Printf("  [dry run] Downloading package %s version %s from %s to %s [artifact=%s]\n",
			pkg.ID, pkg.Version.String(), mr.url.String(), destPath, art.URL)
		return nil
	}

	resp, err := http.Get(art.URL)
	if err != nil {
		return errors.Wrapf(err, "http.Get(%q)", art.URL)
	}
	defer resp.Body.Close() //nolint:errcheck

	outFile, err := os.Create(destPath)
	if err != nil {
		return errors.Wrapf(err, "os.Create(%q)", destPath)
	}
	defer outFile.Close() //nolint:errcheck

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "io.Copy() to %q", destPath)
	}
	return nil
}

func (mr *HTTPRepository) findArtifact(id string, version manifest.SemanticVersion) *manifest.Artifact {
	for pkgID, pkg := range mr.repoConfig.Packages {
		for _, art := range pkg.Artifacts {
			if id == pkgID &&
				art.Version.Major == version.Major &&
				art.Version.Minor == version.Minor &&
				art.Version.Patch == version.Patch {
				return &art
			}
		}
	}
	return nil
}

type MultiRepository struct {
	repos []Repository

	pas []*RepoPackage
}

var (
	_ Repository = (*LocalFileRepository)(nil)
	_ Repository = (*HTTPRepository)(nil)
	_ Repository = (*MultiRepository)(nil)
)

// NewMultiRepository creates a Repository which defers to multiple repositories in the order given.
func NewMultiRepository(repos ...Repository) *MultiRepository {
	return &MultiRepository{
		repos: repos,
	}
}

func (mr *MultiRepository) AddRepository(repo Repository) {
	mr.repos = append(mr.repos, repo)
	mr.pas = nil // invalidate cached packages
}

func (mr *MultiRepository) String() string {
	return fmt.Sprintf("MultiRepository(%v)", mr.repos)
}

func (mr *MultiRepository) ID() string {
	return "<MultiRepository>"
}

func (mr *MultiRepository) DownloadPackage(
	ctx context.Context, pkg *RepoPackage, destPath string, dryRun bool,
) error {
	for _, r := range mr.repos {
		if r.ID() != pkg.RepositoryID {
			continue
		}
		slog.Debug("MultiRepository.DownloadPackage() trying repo", "repo", r.ID(), "for package", pkg.ID)
		return errors.AddStack(r.DownloadPackage(ctx, pkg, destPath, dryRun))
	}
	return fmt.Errorf("package %s not found in any repository", pkg.ID)
}

// FetchPackages fetches each repository and adds their packages to the collection of PackageArtifacts.
func (mr *MultiRepository) FetchPackages(ctx context.Context) ([]*RepoPackage, error) {
	slog.Debug("fetching packages", "repos", mr.repos)
	mr.pas = []*RepoPackage{}
	for _, repo := range mr.repos {
		pas, err := repo.FetchPackages(ctx)
		if err != nil {
			return nil, errors.AddStack(err)
		}
		mr.pas = append(mr.pas, pas...)
	}
	return mr.pas, nil
}

func readJSONFromURL(ctx context.Context, url *url.URL, out interface{}) error {
	var r io.Reader
	switch url.Scheme {
	case "http", "https":
		// TODO: use retryablehttp?
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
		if err != nil {
			return errors.Wrapf(err, "http.NewRequestWithContext(%q)", url.String())
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", version.FullVersion)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrapf(err, "http.Get(%q)", url.String())
		}
		defer func() { _ = resp.Body.Close() }()
		r = resp.Body
	case "file":
		f, err := os.Open(url.Path)
		if err != nil {
			return errors.Wrapf(err, "os.Open(%q)", url.Path)
		}
		defer f.Close()
		r = f
	default:
		return fmt.Errorf("unsupported URL scheme: %s", url.Scheme)
	}

	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		return errors.Wrapf(err, "failed to decode JSON from %s", url.String())
	}
	return nil
}
