package image

import (
	"LGM/filetree"
	"strings"
)

// ShortId returns the truncated id of the current layer.
func (layer *dockerLayer) Command() string {
	return strings.TrimPrefix(layer.history.CreatedBy, "/bin/sh -c ")
}

func (layer *dockerLayer) Tree() *filetree.FileTree{
	return layer.tree
}