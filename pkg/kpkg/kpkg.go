package kpkg

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository/manifest"
	"github.com/pingcap/errors"
	"github.com/ulikunitz/xz"
)

type KPKG struct {
	mu sync.Mutex

	Manifest *manifest.Manifest

	file      *os.File
	tarReader *tar.Reader

	closerFuncs []func() error
}

func Open(path string) (*KPKG, error) {
	kpkg := &KPKG{} //nolint:exhaustruct // this is initialized as we go, to register closers

	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Open(%q)", path)
	}
	kpkg.RegisterCloser(f.Close)
	kpkg.file = f

	var r io.Reader
	r, err = xz.NewReader(f)
	if err != nil {
		slog.Debug("not xz compressed, trying gzip", "error", err)
		_, err = f.Seek(0, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "f.Seek(0,0) for %q", path)
		}
		r, err = gzip.NewReader(f)
		if err != nil {
			slog.Debug("not gzip compressed, using raw file", "error", err)
			_, err = f.Seek(0, 0)
			if err != nil {
				return nil, errors.Wrapf(err, "f.Seek(0,0) for %q", path)
			}
			r = f
		}
	}

	kpkg.tarReader = tar.NewReader(r)

	err = kpkg.ReadMetadata()
	if err != nil {
		cerr := kpkg.Close()
		if cerr != nil {
			slog.Error("ReadMetadata()", "close_error", cerr, "extract_error", err)
		}
		return nil, errors.Wrapf(err, "kpkg.ReadMetadata() for %q", path)
	}

	return kpkg, nil
}

func (k *KPKG) ReadMetadata() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.Manifest != nil {
		return nil
	}

	for {
		entry, err := k.tarReader.Next()
		if err != nil {
			return errors.Wrapf(err, "tarReader.Next()")
		}
		path := entry.Name
		if path == "./" || path == "." {
			continue
		}
		path = strings.TrimPrefix(path, "./")
		if path == "manifest.json" {
			if entry.Typeflag != tar.TypeReg {
				return fmt.Errorf("manifest.json is not a regular file: %v", entry.Typeflag)
			}
			data := make([]byte, entry.Size)
			_, err := io.ReadFull(k.tarReader, data)
			if err != nil {
				return errors.Wrapf(err, "io.ReadFull() for manifest.json")
			}
			var m manifest.Manifest
			err = json.Unmarshal(data, &m)
			if err != nil {
				return errors.Wrapf(err, "json.Unmarshal() to manifest.Manifest")
			}
			k.Manifest = &m
			return nil
		}
	}
}

func (k *KPKG) RegisterCloser(f func() error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.closerFuncs = append(k.closerFuncs, f)
}

func (k *KPKG) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	var err error
	for _, closer := range k.closerFuncs {
		cerr := closer()
		if cerr != nil {
			err = cerr
		}
	}
	return err
}
