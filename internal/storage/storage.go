package storage

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound    = errors.New("storage: not found")
	ErrInvalidKey  = errors.New("storage: invalid key") // tryna escape like ../../../etc
	ErrTooLarge    = errors.New("storage: file too large")
	ErrUnsupported = errors.New("storage: unsupported file type") // disallowed stuff like .exes or invalid files
)

type Driver interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

type Kind string

const (
	Avatar    Kind = "avatars"
	PostImage Kind = "posts/images"
	PostVideo Kind = "posts/videos"
)

type rule struct {
	MaxBytes int64
	Types    map[string]string // sniffed mime -> ext
}

var (
	imageTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	videoTypes = map[string]string{"video/mp4": ".mp4", "video/webm": ".webm"}
)

var rules = map[Kind]rule{
	Avatar:    {MaxBytes: 5 << 20, Types: imageTypes},
	PostImage: {MaxBytes: 15 << 20, Types: imageTypes},
	PostVideo: {MaxBytes: 200 << 20, Types: videoTypes},
}

type Object struct {
	Key         string
	URL         string
	ContentType string
	Size        int64
}

type Storage struct{ d Driver }

func New(d Driver) *Storage                                     { return &Storage{d: d} }
func (s *Storage) Delete(ctx context.Context, key string) error { return s.d.Delete(ctx, key) }
func (s *Storage) URL(key string) string                        { return s.d.URL(key) }

func (s *Storage) Put(ctx context.Context, kind Kind, r io.Reader) (*Object, error) {
	rule, ok := rules[kind]
	if !ok {
		return nil, fmt.Errorf("storage: unknown kind %q", kind)
	}

	// peek the header without consuming it
	br := bufio.NewReaderSize(r, 4096)
	head, _ := br.Peek(512)
	if len(head) == 0 {
		return nil, ErrUnsupported
	}
	ctype := http.DetectContentType(head)
	ext, ok := rule.Types[ctype]
	if !ok {
		return nil, ErrUnsupported
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	key := path.Join(string(kind), time.Now().UTC().Format("2006/01"), id.String()+ext)
	cr := &capReader{r: br, limit: rule.MaxBytes}
	if err := s.d.Put(ctx, key, cr, -1, ctype); err != nil {
		return nil, err
	}
	return &Object{Key: key, URL: s.d.URL(key), ContentType: ctype, Size: cr.n}, nil
}

type capReader struct {
	r     io.Reader
	limit int64
	n     int64
}

func (c *capReader) Read(p []byte) (int, error) {
	if c.n > c.limit {
		return 0, ErrTooLarge
	}
	if remaining := c.limit - c.n + 1; int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.limit {
		return n, ErrTooLarge
	}
	return n, err
}
