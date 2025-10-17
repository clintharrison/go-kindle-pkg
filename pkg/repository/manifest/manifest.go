package manifest

type Dependency struct {
	ID string `json:"id"`
	// RepositoryID restricts the dependency to a specific repository: this should be used sparingly
	RepositoryID *string `json:"repository_id,omitempty"`
	// Min is the minimum required version (inclusive)
	Min *SemanticVersion `json:"min,omitempty"`
	// Max is the maximum supported version (exclusive)
	Max *SemanticVersion `json:"max,omitempty"`
}

type Artifact struct {
	URL           string          `json:"url"`
	Version       SemanticVersion `json:"version"`
	Dependencies  []Dependency    `json:"dependencies,omitempty"`
	SupportedArch []string        `json:"supported_arch,omitempty"`
}

type Package struct {
	ManifestVersion int        `json:"manifest_version"`
	Name            string     `json:"name"`
	Author          string     `json:"author"`
	Description     string     `json:"description"`
	Artifacts       []Artifact `json:"artifacts"`
}

type RepositoryConfig struct {
	Version     int                `json:"manifest_version"`
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Packages    map[string]Package `json:"packages"`
}
