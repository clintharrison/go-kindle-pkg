package kpkg

import (
	"archive/tar"
	"fmt"
	"io"
	"log/slog"
)

func (k *KPKG) ExtractAll(targetDir string) error {
	for {
		entry, err := k.tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		slog.Debug("extracting", "name", entry.Name, "type", entry.Typeflag, "size", entry.Size)
		err = extractEntry(k.tarReader, entry, targetDir)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractEntry(r io.Reader, entry *tar.Header, targetDir string) error {
	switch entry.Typeflag {
	case tar.TypeDir:
		slog.Warn("UNIMPLEMENTED: directory extraction", "name", entry.Name)
		return nil
	case tar.TypeReg:
		slog.Warn("UNIMPLEMENTED: file extraction", "name", entry.Name)
		return nil
	case tar.TypeSymlink:
		slog.Warn("UNIMPLEMENTED: symlink extraction", "name", entry.Name)
		return nil
	default:
		slog.Error("skipping unsupported entry type", "name", entry.Name, "type", entry.Typeflag)
		return fmt.Errorf("unsupported entry type: %v", entry.Typeflag)
	}
}
