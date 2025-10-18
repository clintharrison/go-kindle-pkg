package extract_test

import (
	"bytes"
	_ "embed"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/clintharrison/go-kindle-pkg/cmd/cli/extract"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/koreader_1.2.0_armhf.kpkg
var exampleKpkg []byte

const exampleKpkgFiles = `app/ type=dir mode=755 size=0 uid=1000 gid=100
app/some-bin type=file mode=755 size=62 uid=1000 gid=100
app/legacy-some-bin type=link mode=777 size=0 uid=1000 gid=100 link=some-bin
install.sh type=file mode=755 size=39 uid=1000 gid=100
uninstall.sh type=file mode=755 size=43 uid=1000 gid=100`

func TestExtractCmd_TestArchive(t *testing.T) {
	t.Parallel()

	downloads := t.TempDir()

	koreaderPath := path.Join(downloads, "koreader_1.2.0_armhf.kpkg")
	f, err := os.Create(koreaderPath)
	require.NoError(t, err)
	_, err = f.Write(exampleKpkg)
	require.NoError(t, err)
	f.Close()

	cmd := extract.NewCommand()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	cmd.SetOut(out)
	cmd.SetErr(out)

	err = cmd.ParseFlags([]string{"-t", koreaderPath})
	require.NoError(t, err)

	err = cmd.ParseFlags([]string{"--test", koreaderPath})
	require.NoError(t, err)

	cmd.SetArgs([]string{"-t", koreaderPath})
	err = cmd.Execute()
	require.NoError(t, err)

	require.Equal(
		t,
		strings.TrimSpace(exampleKpkgFiles),
		strings.TrimSpace(out.String()),
		"expected test output to match golden output",
	)
}

func TestExtractCmd_ExtractArchive(t *testing.T) {
	t.Parallel()

	downloads := t.TempDir()

	pkgsPath := path.Join(downloads, "pkgs")
	koreaderPath := path.Join(downloads, "koreader_1.2.0_armhf.kpkg")
	f, err := os.Create(koreaderPath)
	require.NoError(t, err)
	_, err = f.Write(exampleKpkg)
	require.NoError(t, err)
	f.Close()

	cmd := extract.NewCommand()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	cmd.SetOut(out)
	cmd.SetErr(out)

	cmd.SetArgs([]string{"--output", pkgsPath, koreaderPath})
	err = cmd.Execute()
	require.NoError(t, err)

	files, err := os.ReadDir(pkgsPath)
	require.NoError(t, err, "expected extraction to create directory")
	require.NotEmpty(t, files, "expected extracted files to exist in output directory")

	// lazy way to make sure everything got extracted
	extractedFiles := make([]string, 0, len(files))
	for _, file := range files {
		extractedFiles = append(extractedFiles, file.Name())
	}
	require.ElementsMatch(t, extractedFiles, []string{
		"app",
		"install.sh",
		"uninstall.sh",
	})

	files, err = os.ReadDir(pkgsPath + "/app")
	require.NoError(t, err, "expected to be able to read extracted app directory")
	require.NotEmpty(t, files, "expected extracted files to exist in app directory")

	appFiles := make([]string, 0, len(files))
	for _, file := range files {
		appFiles = append(appFiles, file.Name())
	}
	require.ElementsMatch(t, appFiles, []string{
		"some-bin",
		"legacy-some-bin",
	})

	// TODO: check file contents, modes (exec bit!), links, ownership(?)
}
