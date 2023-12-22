package storage

import (
	"sync"

	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

// Seen is used to keep track of all blobs we have already seen across all
// stored images.
type Seen struct {
	mtx  sync.Mutex
	data map[digest.Digest]types.BlobInfo
}

// Add adds a blob to the seen list.
func (s *Seen) Add(blobDigest digest.Digest, blobInfo types.BlobInfo) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.data[blobDigest] = blobInfo
}

// Remove gets a blob info from the seen list.
func (s *Seen) Get(blobDigest digest.Digest) (types.BlobInfo, bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	blobInfo, ok := s.data[blobDigest]
	return blobInfo, ok
}

func NewSeen() *Seen {
	return &Seen{data: map[digest.Digest]types.BlobInfo{}}
}
