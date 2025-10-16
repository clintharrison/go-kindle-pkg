package repository

import (
	"encoding/json"
	"fmt"
)

type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

func (sv *SemanticVersion) UnmarshalJSON(data []byte) error {
	var vs []int
	json.Unmarshal(data, &vs)
	if len(vs) != 3 {
		return fmt.Errorf("semver expected to be [major, minor, patch]")
	}
	sv.Major = vs[0]
	sv.Minor = vs[1]
	sv.Patch = vs[2]
	return nil
}

type Dependency struct {
	ID string `json:"id"`
	// Min is the minimum required version (inclusive)
	Min SemanticVersion `json:"min,omitempty"`
	// Max is the maximum supported version (exclusive)
	Max SemanticVersion `json:"max,omitempty"`
}

type Artifact struct {
	URL           string          `json:"url"`
	Version       SemanticVersion `json:"version"`
	Dependencies  []Dependency    `json:"dependencies,omitempty"`
	SupportedArch []string        `json:"supported_arch,omitempty"`
}

type Package struct {
	ID          string     `json:"id"`
	Version     int        `json:"manifest_version"`
	Name        string     `json:"name"`
	Author      string     `json:"author"`
	Description string     `json:"description"`
	Artifacts   []Artifact `json:"artifacts"`
}

type Repository struct {
	Version     int                `json:"manifest_version"`
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Packages    map[string]Package `json:"packages"`
}
