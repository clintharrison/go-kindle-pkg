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

func NewPackageArtifact(packageID string, repo *manifest.RepositoryConfig, pkg *manifest.Package, art *manifest.Artifact) *PackageArtifact {
	pa := PackageArtifact{
		ID:            packageID,
		RepositoryID:  repo.ID,
		Version:       art.Version,
		SupportedArch: art.SupportedArch,
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

type MultiRepository struct {
	urls []*url.URL

	pas []*PackageArtifact
}

var _ Repository = (*MultiRepository)(nil)

// NewFromURLs creates a MultiRepository from a list of repository URLs.
// Repositories must be a valid file:// or http(s):// URL containing a repository manifest JSON file.
func NewFromURLs(urls ...string) (Repository, error) {
	parsedURLs := make([]*url.URL, len(urls))
	for i, rawurl := range urls {
		parsed, err := url.Parse(rawurl)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", rawurl, err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "file" {
			return nil, fmt.Errorf("invalid URL scheme %q in repo %q", parsed.Scheme, rawurl)
		}
		parsedURLs[i] = parsed
	}
	return &MultiRepository{urls: parsedURLs}, nil
}

// FetchPackages fetches each repository and adds their packages to the collection of PackageArtifacts.
func (mr *MultiRepository) FetchPackages(ctx context.Context) ([]*PackageArtifact, error) {
	mr.pas = []*PackageArtifact{}
	for _, u := range mr.urls {
		var repoConfig manifest.RepositoryConfig
		if err := readJSONFromURL(ctx, u, &repoConfig); err != nil {
			return nil, fmt.Errorf("failed to read repository from %q: %w", u.String(), err)
		}

		for id, pkg := range repoConfig.Packages {
			for _, art := range pkg.Artifacts {
				mr.pas = append(mr.pas, NewPackageArtifact(id, &repoConfig, &pkg, &art))
			}
		}
	}
	return mr.pas, nil
}

func readJSONFromURL(ctx context.Context, u *url.URL, v interface{}) error {
	var r io.Reader
	switch u.Scheme {
	case "http", "https":
		// TODO: use retryablehttp?
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", version.FullVersion)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		r = resp.Body
	case "file":
		f, err := os.Open(u.Path)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	default:
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	return json.NewDecoder(r).Decode(v)
}
