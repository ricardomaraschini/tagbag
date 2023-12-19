package storage

import (
	"bytes"
	"context"
	"os"
	"path"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestNewImages(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)
	tdir := New(tmpdir)
	for _, img := range []string{"img1:latest", "img2:latest"} {
		err := tdir.Image(img)
		assert.NoError(t, err)
	}
	for _, img := range []string{"img1:latest", "img2:latest"} {
		dname := path.Join(tmpdir, img)
		_, err := os.Stat(dname)
		assert.NoError(t, err)
	}
}

func TestPutBlob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tmpdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)
	tdir := New(tmpdir)
	err = tdir.Image("img:latest")
	assert.NoError(t, err)
	dst, err := tdir.NewImageDestination(ctx, nil)
	assert.NoError(t, err)
	content := []byte("testing")
	buf := bytes.NewBuffer(content)
	binfo := types.BlobInfo{
		Size: int64(len(content)),
	}
	binfo, err = dst.PutBlob(ctx, buf, binfo, nil, false)
	assert.NoError(t, err)
	dstw := dst.(*destwrap)
	_, ok := dstw.seen[binfo.Digest]
	assert.True(t, ok)
	blobpath := path.Join(tmpdir, "img:latest", binfo.Digest.Hex())
	stored, err := os.ReadFile(blobpath)
	assert.NoError(t, err)
	if !reflect.DeepEqual(content, stored) {
		t.Errorf("content mismatch: %s -> %s", string(content), string(stored))
	}
}

func TestTryReusingBlob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tmpdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)
	tdir := New(tmpdir)
	err = tdir.Image("img0")
	assert.NoError(t, err)
	dst, err := tdir.NewImageDestination(ctx, nil)
	assert.NoError(t, err)
	content := []byte("testing")
	buf := bytes.NewBuffer(content)
	binfo := types.BlobInfo{
		Digest: digest.FromBytes(content),
		Size:   int64(len(content)),
	}
	reuse, _, err := dst.TryReusingBlob(ctx, binfo, nil, false)
	assert.NoError(t, err)
	assert.False(t, reuse)
	binfo, err = dst.PutBlob(ctx, buf, binfo, nil, false)
	assert.NoError(t, err)
	reuse, _, err = dst.TryReusingBlob(ctx, binfo, nil, false)
	assert.NoError(t, err)
	assert.True(t, reuse)
	err = tdir.Image("img1")
	assert.NoError(t, err)
	reuse, _, err = dst.TryReusingBlob(ctx, binfo, nil, false)
	assert.NoError(t, err)
	assert.True(t, reuse)
}

func Test_findBlob(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)
	for i := 0; i < 10; i++ {
		dirpath := path.Join(tmpdir, strconv.Itoa(i))
		err := os.Mkdir(dirpath, 0755)
		assert.NoError(t, err)
		for j := 0; j < 10; j++ {
			fpath := path.Join(dirpath, strconv.Itoa(j))
			err := os.WriteFile(fpath, []byte("."), 0600)
			assert.NoError(t, err)
		}
	}
	src := &srcwrap{basedir: tmpdir}
	_, err = src.findBlob(digest.FromString("does not exist"))
	assert.Error(t, err)
	dgst := digest.FromString("does exist")
	fpath := path.Join(tmpdir, "9", dgst.Hex())
	err = os.WriteFile(fpath, []byte("test"), 0600)
	assert.NoError(t, err)
	foundpath, err := src.findBlob(dgst)
	assert.NoError(t, err)
	assert.Equal(t, fpath, foundpath)
}

func TestGetBlob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tmpdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)
	tdir := New(tmpdir)
	err = tdir.Image("0")
	assert.NoError(t, err)
	dst, err := tdir.NewImageDestination(ctx, nil)
	assert.NoError(t, err)
	content := []byte("testing")
	buf := bytes.NewBuffer(content)
	binfo := types.BlobInfo{
		Size: int64(len(content)),
	}
	binfo, err = dst.PutBlob(ctx, buf, binfo, nil, false)
	assert.NoError(t, err)
	src, err := tdir.NewImageSource(ctx, nil)
	assert.NoError(t, err)
	fp, _, err := src.GetBlob(ctx, binfo, nil)
	assert.NoError(t, err)
	fp.Close()
	err = tdir.Image("1")
	src, err = tdir.NewImageSource(ctx, nil)
	assert.NoError(t, err)
	fp, _, err = src.GetBlob(ctx, binfo, nil)
	assert.NoError(t, err)
	fp.Close()
}
