package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/clintharrison/go-kindle-pkg/pkg/version"
	"github.com/pingcap/errors"
)

// PackageArtifact is used to uniquely represent a concrete artifact in dependency resolution.
// It does not need to fully duplicate all user-facing metadata or package URLs, just enough
// to construct the dependency graph.
type PackageArtifact struct {
	ID            string
	RepositoryID  string
	Version       manifest.SemanticVersion
	SupportedArch []string
	Dependencies  []PackageDependency
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

func NewPackageArtifact(
	packageID string, repo *manifest.RepositoryConfig, art *manifest.Artifact,
) *PackageArtifact {
	pa := PackageArtifact{
		ID:            packageID,
		RepositoryID:  repo.ID,
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

type Repository interface {
	FetchPackages(ctx context.Context) ([]*PackageArtifact, error)
}

type HTTPRepository struct {
	url *url.URL
	pas []*PackageArtifact
}

func (mr *HTTPRepository) FetchPackages(ctx context.Context) ([]*PackageArtifact, error) {
	mr.pas = []*PackageArtifact{}
	var repoConfig manifest.RepositoryConfig
	err := readJSONFromURL(ctx, mr.url, &repoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository from %q: %w", mr.url.String(), err)
	}

	for id, pkg := range repoConfig.Packages {
		for _, art := range pkg.Artifacts {
			mr.pas = append(mr.pas, NewPackageArtifact(id, &repoConfig, &art))
		}
	}
	return mr.pas, nil
}

type MultiRepository struct {
	repos []Repository

	pas []*PackageArtifact
}

var (
	_ Repository = (*HTTPRepository)(nil)
	_ Repository = (*MultiRepository)(nil)
)

// NewFromURLs creates a MultiRepository from a list of repository URLs.
// Repositories must be a valid file:// or http(s):// URL containing a repository manifest JSON file.
func NewFromURLs(urls ...string) (*MultiRepository, error) {
	repos := make([]Repository, len(urls))
	for i, rawurl := range urls {
		parsed, err := url.Parse(rawurl)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", rawurl, err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "file" {
			return nil, fmt.Errorf("invalid URL scheme %q in repo %q", parsed.Scheme, rawurl)
		}
		repos[i] = &HTTPRepository{url: parsed, pas: nil}
	}
	return &MultiRepository{repos: repos, pas: nil}, nil
}

// FetchPackages fetches each repository and adds their packages to the collection of PackageArtifacts.
func (mr *MultiRepository) FetchPackages(ctx context.Context) ([]*PackageArtifact, error) {
	mr.pas = []*PackageArtifact{}
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
