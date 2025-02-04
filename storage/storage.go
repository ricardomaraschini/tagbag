package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

// Storage handles the storage of multiple images. It is used to store multiple
// images in a single directory while deduplicating blobs.
type Storage struct {
	types.ImageReference
	seen    *Seen
	curimg  string
	basedir string
}

// New returns a reference to a Storage using provided directory as base (root).
// Property "seen" is used to we keep track of all blobs we have already seen
// across all stored images.
func New(basedir string) *Storage {
	return &Storage{
		basedir: basedir,
		seen:    NewSeen(),
	}
}

// CurrentImage returns the inner image we are operating on.
func (t *Storage) CurrentImage() string {
	if t.ImageReference == nil {
		return ""
	}
	return t.curimg
}

// Images list all images stored in the Storage. Traverses the Storage base
// directory and returns all subdirectories that do not contain a subdir or
// name starts with ".".
func (t *Storage) Images() ([]string, error) {
	var images []string
	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if !d.IsDir() {
			return nil
		}
		dirname := filepath.Base(d.Name())
		if strings.HasPrefix(dirname, ".") {
			return filepath.SkipDir
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				return nil
			}
		}
		image, err := filepath.Rel(t.basedir, path)
		if err != nil {
			return fmt.Errorf("failed to get rel path: %w", err)
		}
		images = append(images, image)
		return nil
	}
	if err := filepath.WalkDir(t.basedir, walker); err != nil {
		return nil, fmt.Errorf("fail to traverse storage: %w", err)
	}
	return images, nil
}

// DeleteBlob deletes a blob from the current Storage. The file name must be
// a blob file name, i.e. a file name that is a valid digest.
func (t *Storage) DeleteBlob(name string) error {
	matches, _ := regexp.MatchString("^[a-fA-F0-9]{64}$", name)
	if !matches {
		return nil
	}
	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != name {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove blob: %w", err)
		}
		dirname := filepath.Dir(path)
		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("failed to read dir: %w", err)
		}
		if len(entries) != 0 {
			return nil
		}
		if err := os.Remove(dirname); err != nil {
			return fmt.Errorf("failed to remove dir: %w", err)
		}
		return nil
	}
	if err := filepath.WalkDir(t.basedir, walker); err != nil {
		return fmt.Errorf("fail to traverse storage: %w", err)
	}
	return nil
}

// Files returns a list of all files stored in the Storage, includes files in
// all Images.
func (t *Storage) Files() (map[string]os.FileInfo, error) {
	images := map[string]os.FileInfo{}
	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if d.IsDir() {
			return nil
		}
		dirname := filepath.Base(d.Name())
		if strings.HasPrefix(dirname, ".") {
			return filepath.SkipDir
		}
		relpath, err := filepath.Rel(t.basedir, path)
		if err != nil {
			return fmt.Errorf("failed to get rel path: %w", err)
		}
		stat, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}
		images[relpath] = stat
		return nil
	}
	if err := filepath.WalkDir(t.basedir, walker); err != nil {
		return nil, fmt.Errorf("fail to traverse storage: %w", err)
	}
	return images, nil
}

// Image sets the current inner Image inside the Storage. A Storage allows
// "write to" and "read from" on a single Image at a given time. Creates or
// uses an already existent subdirectory named after the Image name.
func (t *Storage) Image(image string) error {
	gendir := path.Join(t.basedir, image)
	if _, err := os.Stat(gendir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat dir: %w", err)
		}
		if err := os.MkdirAll(gendir, 0700); err != nil {
			return fmt.Errorf("failed to create dir: %w", err)
		}
	}
	// Here we enable a regular containers/image Directory transport for
	// the Image subdirectory we are about to start dealing with.
	inref, err := directory.NewReference(gendir)
	if err != nil {
		return fmt.Errorf("failed to create dir ref: %w", err)
	}
	t.ImageReference = inref
	t.curimg = image
	return nil
}

// NewImageSource returns a handler used to read from the Storage current
// Image. Current image must already be set by caller by means of a call
// to Image function.
func (t *Storage) NewImageSource(
	ctx context.Context, sys *types.SystemContext,
) (types.ImageSource, error) {
	src, err := t.ImageReference.NewImageSource(ctx, sys)
	if err != nil {
		return nil, err
	}
	return &srcwrap{basedir: t.basedir, ImageSource: src}, nil
}

// NewImageDestination returns a handler used to write to the current
// Image. Current image must be already set by the caller (using Image()
// function).
func (t *Storage) NewImageDestination(
	ctx context.Context, sys *types.SystemContext,
) (types.ImageDestination, error) {
	dst, err := t.ImageReference.NewImageDestination(ctx, sys)
	if err != nil {
		return nil, err
	}
	return &destwrap{
		ImageDestination: dst,
		seen:             t.seen,
		image:            t.curimg,
	}, nil
}

// destwrap wraps an ImageDestination and adds a check for already
// pulled blobs. Already pulled blobs are kept on "seen" property.
// As DirectoryTransport does not support concurrent access I did
// not take any special care here regarding the usage of "seen"
// property. XXX sync.Mutex, alstublieft.
type destwrap struct {
	types.ImageDestination
	image string
	seen  *Seen
}

// PutBlob calls underlying ImageDestination PutBlob function and
// if the call succeeds it register the blob as already seen.
// Package containers/image access the TryReusingBlob before this
// one so we do the cache check there.
func (d *destwrap) PutBlob(
	ctx context.Context,
	stream io.Reader,
	info types.BlobInfo,
	cache types.BlobInfoCache,
	iscfg bool,
) (types.BlobInfo, error) {
	binfo, err := d.ImageDestination.PutBlob(
		ctx, stream, info, cache, iscfg,
	)
	if err != nil {
		return binfo, err
	}
	d.seen.Add(binfo.Digest, binfo)
	return binfo, nil
}

// TryReusingBlob checks if a blob has already been "seen", pulled.
// If yes then returns true informing that we can "reuse" the blob.
// With that containers/image won't attempt to pull the blob thus
// calling PutBlob.
func (d *destwrap) TryReusingBlob(
	ctx context.Context,
	info types.BlobInfo,
	cache types.BlobInfoCache,
	substitute bool,
) (bool, types.BlobInfo, error) {
	if binfo, ok := d.seen.Get(info.Digest); ok {
		return true, binfo, nil
	}
	return d.ImageDestination.TryReusingBlob(
		ctx, info, cache, substitute,
	)
}

// srcwrap is a wrap around a ImageSource interface. It is specifically
// designed to leverage blobs already pulled in any of the existing
// Images inside a Storage. This wraps searchs for Digest named files
// from "basedir" inwards.
type srcwrap struct {
	basedir string
	types.ImageSource
}

// findBlob attempts to find a blob in any of the already pulled Images.
// Returns either the blob path or an error. XXX this should be optimized
// to avoid reading all blobs every time.
func (s *srcwrap) findBlob(dgst digest.Digest) (string, error) {
	var blobpath string
	walker := func(path string, info fs.FileInfo, err error) error {
		if info.Name() == dgst.Hex() {
			blobpath = path
		}
		return nil
	}
	if err := filepath.Walk(s.basedir, walker); err != nil {
		return "", fmt.Errorf("fail to traverse storage: %w", err)
	}
	if len(blobpath) == 0 {
		return "", fmt.Errorf("blob %s not found", dgst.Hex())
	}
	return blobpath, nil
}

// GetBlob attempts to return a reader for a blob file in a given Image.
// This function first tries to check if the blob is present in the current
// Image, if it is not then searches for it in all other images.
func (s *srcwrap) GetBlob(
	ctx context.Context, info types.BlobInfo, icache types.BlobInfoCache,
) (io.ReadCloser, int64, error) {
	stream, size, err := s.ImageSource.GetBlob(ctx, info, icache)
	if err == nil {
		return stream, size, nil
	} else if !os.IsNotExist(err) {
		return stream, size, err
	}
	blobpath, err := s.findBlob(info.Digest)
	if err != nil {
		return nil, -1, err
	}
	fp, err := os.Open(blobpath)
	if err != nil {
		return nil, -1, err
	}
	fi, err := fp.Stat()
	if err != nil {
		return nil, -1, err
	}
	return fp, fi.Size(), nil
}
