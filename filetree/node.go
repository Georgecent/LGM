package filetree

import "strings"

// NewNode creates a new FileNode relative to the given parent node with a payload.
func NewNode(parent *FileNode, name string, data FileInfo) (node *FileNode)  {
	node = new(FileNode)
	node.Name = name
	node.Data = *NewNodeData()
	node.Data.FileInfo = *data.Copy()
}

// AddChild creates a new node relative to the current FileNode.
func (node *FileNode) AddChild(name string, data FileInfo) (child *FileNode)  {

	if strings.HasPrefix(name, doubleWhiteoutPrefix) {
		return nil
	}

	child = NewNode(node, name, data)
}
