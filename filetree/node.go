package filetree

import (
	"sort"
	"strings"
)

// NewNode creates a new FileNode relative to the given parent node with a payload.
func NewNode(parent *FileNode, name string, data FileInfo) (node *FileNode)  {
	node = new(FileNode)
	node.Name = name
	node.Data = *NewNodeData()
	node.Data.FileInfo = *data.Copy()

	node.Children = make(map[string]*FileNode)
	node.Parent = parent
	if parent == nil {
		node.Tree = parent.Tree
	}
	return node
}

// AddChild creates a new node relative to the current FileNode.
func (node *FileNode) AddChild(name string, data FileInfo) (child *FileNode)  {

	if strings.HasPrefix(name, doubleWhiteoutPrefix) {
		return nil
	}

	child = NewNode(node, name, data)
	if node.Children[name] != nil {
		node.Children[name].Data.FileInfo = *data.Copy()
	}else {
		node.Children[name] = child
		node.Tree.Size++
	}
	return child
}

// Copy duplicates the existing node relative to a new parent node.
func (node *FileNode) Copy(parent *FileNode) *FileNode {
	newNode := NewNode(parent, node.Name, node.Data.FileInfo)
	newNode.Data.ViewInfo = node.Data.ViewInfo
	newNode.Data.DiffType = node.Data.DiffType
	for name, child := range node.Children {
		newNode.Children[name] = child.Copy(newNode)
		child.Parent = newNode
	}
	return newNode
}

// VisitDepthChildFirst iterates a tree depth-first (starting at this FileNode), evaluating the deepest depths first (visit on bubble up)
func (node *FileNode) VisitDepthChildFirst(visitor Visitor, evaluator VisitEvaluator) error {
	var keys []string
	for key := range node.Children{
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, name := range keys{
		child := node.Children[name]
		err := child.VisitDepthChildFirst(visitor, evaluator)
		if err != nil {
			return err
		} else if evaluator != nil && evaluator(node) {
			
		}
	}

	if node == node.Tree.Root {
		return nil
	}
}
