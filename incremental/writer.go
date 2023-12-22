package incremental

import (
	"context"
	"fmt"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

// Writer provides a tool to copy only the layers that are not already
// present in a different version of the same image.
type Writer struct {
	types.ImageReference
	dest *destwrap
}

// NewImageDestination returns a handler used to write.
func (i *Writer) NewImageDestination(ctx context.Context, sys *types.SystemContext) (types.ImageDestination, error) {
	return i.dest, nil
}

// destwrap wraps an image destination (this can be another registry or a
// file on disk) and the original manifest from where we can extract the
// layers that are already present.
type destwrap struct {
	types.ImageDestination
	manifests map[digest.Digest]bool
}

// TryReusingBlob is called by the image copy code to check if a layer is
// already present in the destination. If it is, we return true and the
// layer info. If it is not, we return false and the layer info. We use the
// manifest to check if the layer is already present.
func (d *destwrap) TryReusingBlob(ctx context.Context, info types.BlobInfo, cache types.BlobInfoCache, substitute bool) (bool, types.BlobInfo, error) {
	if _, ok := d.manifests[info.Digest]; ok {
		return true, info, nil
	}
	return false, info, nil
}

// processList expects raw to point to a manifest list and will iterate over
// the children and return them.
func processList(ctx context.Context, fromref types.ImageSource, raw []byte, mime string) ([]manifest.Manifest, error) {
	list, err := manifest.ListFromBlob(raw, mime)
	if err != nil {
		return nil, fmt.Errorf("error parsing manifests: %w", err)
	}
	children := []manifest.Manifest{}
	for _, digest := range list.Instances() {
		raw, mime, err := fromref.GetManifest(ctx, &digest)
		if err != nil {
			return nil, fmt.Errorf("error getting child manifest: %w", err)
		}
		man, err := manifest.FromBlob(raw, mime)
		if err != nil {
			return nil, fmt.Errorf("error parsing manifest: %w", err)
		}
		children = append(children, man)
	}
	return children, nil
}

// fetchManifests returns the list of manifests that are present in the
// source image. In case of a manifest list it will iterate over the
// children and return them.
func fetchManifests(ctx context.Context, from types.ImageReference, sysctx *types.SystemContext) ([]manifest.Manifest, error) {
	fromref, err := from.NewImageSource(ctx, sysctx)
	if err != nil {
		return nil, fmt.Errorf("error creating image source: %w", err)
	}
	defer fromref.Close()
	raw, mime, err := fromref.GetManifest(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting manifest: %w", err)
	}
	if manifest.MIMETypeIsMultiImage(mime) {
		return processList(ctx, fromref, raw, mime)
	}
	man, err := manifest.FromBlob(raw, mime)
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest: %w", err)
	}
	return []manifest.Manifest{man}, nil
}

// NewWriter is capable of providing an incremental copy of an image using
// 'from' as base and storing the result in 'to'.
func NewWriter(ctx context.Context, from types.ImageReference, to types.ImageReference, sysctx *types.SystemContext) (*Writer, error) {
	toref, err := to.NewImageDestination(ctx, sysctx)
	if err != nil {
		return nil, fmt.Errorf("error creating destination: %w", err)
	}
	manifests, err := fetchManifests(ctx, from, sysctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching manifests: %w", err)
	}
	mandict := map[digest.Digest]bool{}
	for _, man := range manifests {
		for _, layer := range man.LayerInfos() {
			mandict[layer.Digest] = true
		}
	}
	return &Writer{
		ImageReference: to,
		dest: &destwrap{
			ImageDestination: toref,
			manifests:        mandict,
		},
	}, nil
}
