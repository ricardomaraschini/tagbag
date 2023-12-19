package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

// Commiter implements commit and commit is called everytime a blob is
// saved to the Storage.
type Committer interface {
	Commit() error
}

type Storage struct {
	types.ImageReference
	seen     map[digest.Digest]types.BlobInfo
	curimg   string
	basedir  string
	commiter Committer
}

// New returns a reference to a Storage using provided directory as base
// (root). Property "seen" is used to we keep track of all blobs we have
// already seen (pulled) across all stored images.
func New(basedir string, committer Committer) *Storage {
	return &Storage{
		basedir:  basedir,
		seen:     map[digest.Digest]types.BlobInfo{},
		commiter: committer,
	}
}

// CurrentImage returns the inner image we are operating on.
func (t *Storage) CurrentImage() string {
	if t.ImageReference == nil {
		return ""
	}
	return t.curimg
}

// Image sets the current inner Image inside the Storage. A Storage allows
// "write to" and "read from" on a single Image at a given time. Creates or
// uses an already existent subdirectory named after the Image name.
func (t *Storage) Image(image string) error {
	gendir := path.Join(t.basedir, image)
	if _, err := os.Stat(gendir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(gendir, 0700); err != nil {
			return err
		}
	}
	// Here we enable a regular containers/image Directory transport for
	// the Image subdirectory we are about to start dealing with.
	inref, err := directory.NewReference(gendir)
	if err != nil {
		return err
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
	return &srcwrap{
		basedir:     t.basedir,
		ImageSource: src,
	}, nil
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
		committer:        t.commiter,
	}, nil
}

// destwrap wraps an ImageDestination and adds a check for already
// pulled blobs. Already pulled blobs are kept on "seen" property.
// As DirectoryTransport does not support concurrent access I did
// not take any special care here regarding the usage of "seen"
// property. XXX sync.Mutex, alstublieft.
type destwrap struct {
	types.ImageDestination
	seen      map[digest.Digest]types.BlobInfo
	committer Committer
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
	d.seen[binfo.Digest] = binfo
	return binfo, nil
}

// Close is called when the write to the Image is finished. It
// calls the underlying committer function so we now it is time
// to commit the changes and close the underlying dest Image.
func (d *destwrap) Close() error {
	if err := d.committer.Commit(); err != nil {
		return err
	}
	return d.ImageDestination.Close()
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
	if binfo, ok := d.seen[info.Digest]; ok {
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
	if err := filepath.Walk(
		s.basedir,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == dgst.Hex() {
				blobpath = path
			}
			return nil
		},
	); err != nil {
		return "", err
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
