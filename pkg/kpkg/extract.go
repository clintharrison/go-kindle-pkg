package kpkg

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/pingcap/errors"
)

func (k *KPKG) ExtractAll(ctx context.Context, targetDir string, test bool, w io.Writer) error {
	if targetDir == "" {
		return fmt.Errorf("no target directory specified")
	}
	if _, err := os.Stat(targetDir); err != nil {
		if test {
			slog.Info("would create output directory", "path", targetDir)
		} else {
			err := os.MkdirAll(targetDir, 0o755)
			if err != nil {
				return errors.Wrapf(err, "os.MkdirAll(%q)", targetDir)
			}
		}
	}
	for {
		entry, err := k.tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		slog.Debug("extracting", "name", entry.Name, "type", entry.Typeflag, "size", entry.Size)
		if test {
			err := logEntry(w, entry)
			if err != nil {
				return err
			}
		} else {
			err := extractEntry(ctx, w, k.tarReader, entry, targetDir)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// this is a slightly nicer-to-read order than random
var attrOrder = []string{
	"type", "mode", "size", "uid", "gid", "link",
}

func logEntry(w io.Writer, entry *tar.Header) error {
	// TODO: normalize name, don't allow path traversal upwards?
	n := strings.TrimPrefix(entry.Name, "./")
	// replace whitespace with octal escapes.
	// this should maybe replace more than that, but for now this will do?
	for _, r := range []rune{'\t', '\n', '\v', '\f', '\r'} {
		n = strings.ReplaceAll(n, string(r), fmt.Sprintf("\\%o", r))
	}
	w.Write([]byte(n))

	attrs := make(map[string]string)
	attrs["size"] = fmt.Sprintf("%d", entry.Size)
	attrs["mode"] = fmt.Sprintf("%o", entry.Mode)
	attrs["uid"] = fmt.Sprintf("%d", entry.Uid)
	attrs["gid"] = fmt.Sprintf("%d", entry.Gid)

	switch entry.Typeflag {
	case tar.TypeDir:
		attrs["type"] = "dir"
	case tar.TypeReg:
		attrs["type"] = "file"
	case tar.TypeLink:
		fallthrough
	case tar.TypeSymlink:
		attrs["type"] = "link"
		attrs["link"] = entry.Linkname
	case tar.TypeChar:
		attrs["type"] = "char"
	case tar.TypeBlock:
		attrs["type"] = "block"
	case tar.TypeFifo:
		attrs["type"] = "fifo"
	default:
		slog.Error("UNSUPPORTED", "name", entry.Name, "type", entry.Typeflag)
		return fmt.Errorf("unsupported entry type: %v", entry.Typeflag)
	}

	for _, k := range attrOrder {
		v, ok := attrs[k]
		if ok {
			w.Write([]byte(fmt.Sprintf(" %s=%s", k, v)))
		}
	}
	w.Write([]byte("\n"))
	return nil
}

func extractEntry(ctx context.Context, w io.Writer, r io.Reader, entry *tar.Header, targetDir string) error {
	logEntry(w, entry)

	path := strings.TrimPrefix(path.Clean(entry.Name), "./")
	fullPath := targetDir + "/" + path

	switch entry.Typeflag {
	case tar.TypeDir:
		err := os.MkdirAll(fullPath, os.FileMode(entry.Mode))
		if err != nil {
			return errors.Wrapf(err, "os.MkdirAll(%q)", fullPath)
		}
		return nil
	case tar.TypeReg:
		file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(entry.Mode))
		if err != nil {
			return errors.Wrapf(err, "os.OpenFile(%q)", fullPath)
		}
		defer file.Close()
		_, err = io.Copy(file, r)
		if err != nil {
			return errors.Wrapf(err, "io.Copy(%q)", fullPath)
		}
		return nil
	case tar.TypeSymlink:
		err := os.Symlink(entry.Linkname, fullPath)
		if err != nil {
			return errors.Wrapf(err, "os.Symlink(%q, %q)", entry.Linkname, fullPath)
		}
		return nil
	case tar.TypeLink:
		err := os.Link(entry.Linkname, fullPath)
		if err != nil {
			return errors.Wrapf(err, "os.Link(%q, %q)", entry.Linkname, fullPath)
		}
		return nil
	default:
		return fmt.Errorf("package archive has unsupported entry type: %v", entry.Typeflag)
	}
}
