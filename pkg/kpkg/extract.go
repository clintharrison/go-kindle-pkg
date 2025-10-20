package kpkg

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pingcap/errors"
)

func (k *KPKG) ExtractAll(ctx context.Context, targetDir string, test bool, logw io.Writer) error {
	if targetDir == "" {
		return errors.New("no target directory specified")
	}
	_, err := os.Stat(targetDir)
	if err != nil {
		if test {
			slog.Info("would create output directory", "path", targetDir)
		} else {
			err := os.MkdirAll(targetDir, 0o755) //nolint:gosec
			if err != nil {
				return errors.Wrapf(err, "os.MkdirAll(%q)", targetDir)
			}
		}
	}
	err = k.resetReader()
	if err != nil {
		return errors.Wrap(err, "kpkg.resetReader()")
	}

	for {
		entry, err := k.tarReader.Next()
		nm := "<nil>"
		if entry != nil {
			nm = entry.Name
		}
		slog.Debug("Read tar entry", "name", nm, "err", err)
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "tarReader.Next()")
		}
		slog.Debug("extracting", "name", entry.Name, "type", entry.Typeflag, "size", entry.Size, "dest", targetDir)
		if test {
			err := logEntry(logw, entry)
			if err != nil {
				return err
			}
		} else {
			err := extractEntry(ctx, logw, k.tarReader, entry, targetDir)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// attrOrder matches the order from mtree output, which is slightly nicer to read than random.
var attrOrder = []string{ //nolint:gochecknoglobals
	"type", "mode", "size", "uid", "gid", "link",
}

func logEntry(logw io.Writer, entry *tar.Header) error {
	// TODO: normalize name, don't allow path traversal upwards?
	n := strings.TrimPrefix(entry.Name, "./")
	// replace whitespace with octal escapes.
	// this should maybe replace more than that, but for now this will do?
	for _, r := range []rune{'\t', '\n', '\v', '\f', '\r'} {
		n = strings.ReplaceAll(n, string(r), fmt.Sprintf("\\%o", r))
	}
	logw.Write([]byte(n)) //nolint:errcheck

	attrs := make(map[string]string)
	attrs["size"] = strconv.FormatInt(entry.Size, 10)
	attrs["mode"] = fmt.Sprintf("%o", entry.Mode)
	attrs["uid"] = strconv.Itoa(entry.Uid)
	attrs["gid"] = strconv.Itoa(entry.Gid)

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
			fmt.Fprintf(logw, " %s=%s", k, v) //nolint:errcheck
		}
	}
	logw.Write([]byte("\n")) //nolint:errcheck
	return nil
}

func extractEntry(_ context.Context, logw io.Writer, r io.Reader, entry *tar.Header, targetDir string) error {
	// _ = logEntry(logw, entry)

	path := strings.TrimPrefix(path.Clean(entry.Name), "./")
	fullPath := targetDir + "/" + path

	switch entry.Typeflag {
	case tar.TypeDir:
		err := os.MkdirAll(fullPath, os.FileMode(entry.Mode)) //nolint:gosec
		if err != nil {
			return errors.Wrapf(err, "os.MkdirAll(%q)", fullPath)
		}
		return nil
	case tar.TypeReg:
		file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(entry.Mode)) //nolint:gosec
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
