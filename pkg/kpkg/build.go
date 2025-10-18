package kpkg

import (
	"archive/tar"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pingcap/errors"
	"github.com/ulikunitz/xz"
)

type BuildOption func(*options) error

type options struct {
	compressor func(io.WriteCloser) (io.WriteCloser, error)
}

func WithXZCompression(opts *options) error {
	opts.compressor = func(w io.WriteCloser) (io.WriteCloser, error) {
		return xz.NewWriter(w)
	}
	return nil
}

func Build(_ context.Context, rootPath string, dest string, optFuncs ...BuildOption) error {
	opts := &options{} //nolint:exhaustruct
	for _, o := range optFuncs {
		err := o(opts)
		if err != nil {
			return errors.Wrap(err, "applying build option")
		}
	}
	df, err := os.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "opening destination file %q", dest)
	}
	defer df.Close()

	var compressw io.WriteCloser
	if opts.compressor != nil {
		compressw, err = opts.compressor(df)
		if err != nil {
			return errors.Wrap(err, "creating compressor")
		}
		defer compressw.Close() //nolint:errcheck
	} else {
		compressw = df
	}

	tw := tar.NewWriter(compressw)
	defer tw.Close() //nolint:errcheck

	manifestPath := filepath.Join(rootPath, "manifest.json")
	_, err = os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// TODO: create the manifest.json
			return errors.New("manifest.json must be present in the package directory")
		}
		return errors.Wrap(err, "os.Stat(\"manifest.json\")")
	}

	// TODO: Refactor this once Go 1.25 is available, to use tar.NewWriter.AddFS directly?
	// I'm not sure how to mask the uid/timestamps though...
	err = filepath.WalkDir(rootPath, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		linkTarget := ""
		if typ := d.Type(); typ == fs.ModeSymlink {
			var err error
			linkTarget, err = os.Readlink(name)
			if err != nil {
				return err
			}
		} else if !typ.IsRegular() && typ != fs.ModeDir {
			return errors.New("tar: cannot add non-regular file")
		}
		h, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}

		h, err = normalizeHeader(h, info, rootPath, name)
		if err != nil {
			return errors.Wrap(err, "normalizing tar header")
		}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "walking root fs")
	}
	return nil
}

func normalizeHeader(h *tar.Header, d fs.FileInfo, rootPath, name string) (*tar.Header, error) {
	// Make path relative to rootPath
	relPath, err := filepath.Rel(rootPath, name)
	if err != nil {
		return nil, err
	}
	if relPath == "." {
		h.Name = "./"
	} else {
		h.Name = "./" + relPath
	}

	// Normalize metadata; no need to leak host info
	h.Uid = 0
	h.Gid = 0
	h.Uname = ""
	h.Gname = ""
	h.Mode = int64(h.Mode & 0o777)
	h.ModTime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	h.Format = tar.FormatGNU

	// Only regular files have size
	if d.Mode().IsRegular() {
		h.Size = d.Size()
	} else if d.Mode().IsDir() {
		h.Size = 0
		h.Name = strings.TrimSuffix(h.Name, "/") + "/"
	}

	return h, nil
}
