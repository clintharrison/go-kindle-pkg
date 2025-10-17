package testdata

import (
	_ "embed"
)

//go:embed repository.json
var RepositoryJSON []byte

//go:embed koreader_manifest.json
var KoreaderManifestJSON []byte
