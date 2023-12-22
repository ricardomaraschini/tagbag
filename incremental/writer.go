package incremental

import (
	"context"
	"fmt"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
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
	manifest manifest.Manifest
}

// TryReusingBlob is called by the image copy code to check if a layer is
// already present in the destination. If it is, we return true and the
// layer info. If it is not, we return false and the layer info. We use the
// manifest to check if the layer is already present.
func (d *destwrap) TryReusingBlob(ctx context.Context, info types.BlobInfo, cache types.BlobInfoCache, substitute bool) (bool, types.BlobInfo, error) {
	for _, layer := range d.manifest.LayerInfos() {
		if layer.Digest == info.Digest {
			return true, info, nil
		}
	}
	return false, info, nil
}

// NewWriter is capable of providing an incremental copy of an image using
// 'from' as base and storing the result in 'to'.
func NewWriter(ctx context.Context, from types.ImageReference, to types.ImageReference, sysctx *types.SystemContext) (*Writer, error) {
	fromref, err := from.NewImageSource(ctx, sysctx)
	if err != nil {
		return nil, fmt.Errorf("error creating source: %w", err)
	}
	raw, mime, err := fromref.GetManifest(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting manifest: %w", err)
	}
	if manifest.MIMETypeIsMultiImage(mime) {
		return nil, fmt.Errorf("multi-image manifests not supported")
	}
	man, err := manifest.FromBlob(raw, mime)
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest: %w", err)
	}
	toref, err := to.NewImageDestination(ctx, sysctx)
	if err != nil {
		return nil, fmt.Errorf("error creating destination: %w", err)
	}
	return &Writer{
		ImageReference: to,
		dest: &destwrap{
			ImageDestination: toref,
			manifest:         man,
		},
	}, nil
}
