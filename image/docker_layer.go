package image

import (
	"LGM/filetree"
	"fmt"
	"github.com/dustin/go-humanize"
	"strings"
)

const (
	// LayerFormat = "%-15s %7s  %s"
	LayerFormat = "%7s  %s"
)

// ShortId returns the truncated id of the current layer.
func (layer *dockerLayer) Command() string {
	return strings.TrimPrefix(layer.history.CreatedBy, "/bin/sh -c ")
}

// Tree returns the file tree representing the current layer.
func (layer *dockerLayer) Tree() *filetree.FileTree{
	return layer.tree
}

// Size returns the number of bytes that this image is.
func (layer *dockerLayer) Size() uint64 {
	return layer.history.Size
}

// ShortId returns the truncated id of the current layer.
func (layer *dockerLayer) TarId() string {
	return strings.TrimSuffix(layer.tarPath, "/layer.tar")
}

// ShortId returns the truncated id of the current layer.
func (layer *dockerLayer) Id() string {
	return layer.history.ID
}

// index returns the relative position of the layer within the image.
func (layer *dockerLayer) Index() int {
	return layer.index
}

// ShortId returns the truncated id of the current layer.
func (layer *dockerLayer) ShortId() string {
	rangeBound := 15
	id := layer.Id()
	if length := len(id); length < 15 {
		rangeBound = length
	}
	id = id[0:rangeBound]
	return id
}

// String represents a layer in a columnar format.
func (layer *dockerLayer) String() string {

	if layer.index == 0 {
		return fmt.Sprintf(LayerFormat,
			humanize.Bytes(layer.Size()),
			"FROM "+layer.ShortId())
	}
	return fmt.Sprintf(LayerFormat,
		humanize.Bytes(layer.Size()),
		layer.Command())
}