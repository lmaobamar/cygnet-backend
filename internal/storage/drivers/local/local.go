package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmaobamar/cygnet-backend/internal/storage"
)

type Driver struct {
	base   string
	public string // url prefix like "/media"
}

func New(base, public string) (*Driver, error) {
	absolute, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	if public == "" {
		public = "/media"
	}
	return &Driver{base: absolute, public: strings.TrimRight(public, "/")}, nil
}

func (d *Driver) resolvePath(key string) (string, error) {
	if key == "" || strings.ContainsRune(key, 0) {
		return "", storage.ErrInvalidKey
	}
	p := filepath.Join(d.base, filepath.FromSlash(key))
	if !strings.HasPrefix(p, d.base+string(filepath.Separator)) {
		return "", storage.ErrInvalidKey
	}
	return p, nil
}

func (driver *Driver) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	filePath, err := driver.resolvePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	// local will write to a temp file so u dont get half writes
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, reader); err != nil {
		tempFile.Close()
		return err // includes storage.ErrTooLarge from capreader
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempFile.Name(), filePath)
}
