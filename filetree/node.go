package filetree

import (
	"fmt"
	"sort"
	"strings"
)

// IsWhiteout returns an indication if this file may be a overlay-whiteout file.
func (node *FileNode) IsWhiteout() bool {
	return strings.HasPrefix(node.Name, whiteoutPrefix)
}

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
		}
	}

	if node == node.Tree.Root {
		return nil
	}else if evaluator != nil && evaluator(node) || evaluator == nil{
		return visitor(node)
	}
	return nil
}

// Path 返回从较大树的根到当前节点的斜杠分隔字符串(e.g. /a/path/to/here)
func (node *FileNode) Path() string {
	if node.path == "" {
		var path []string
		curNode := node
		for {
			if curNode.Parent == nil {
				break
			}

			name := curNode.Name
			if curNode == node {
				// white out prefixes are fictitious on leaf nodes
				name = strings.TrimPrefix(name, whiteoutPrefix)
			}

			path = append([]string{name}, path...)
			curNode = curNode.Parent
		}
		node.path = "/" + strings.Join(path, "/")
	}
	return strings.Replace(node.path, "//", "/", -1)
}

// Remove deletes the current FileNode from it's parent FileNode's relations.
func (node *FileNode) Remove() error {
	if node == node.Tree.Root {
		return fmt.Errorf("cannot remove the tree root")
	}
	for _, child := range node.Children {
		child.Remove()
	}
	delete(node.Parent.Children, node.Name)
	node.Tree.Size--
	return nil
}

// compare the current node against the given node, returning a definitive DiffType.
func (node *FileNode) compare(other *FileNode) DiffType {
	if node == nil && other == nil {
		return Unchanged
	}

	if node == nil && other != nil {
		return Added
	}

	if node != nil && other == nil {
		return Removed
	}

	if other.IsWhiteout() {
		return Removed
	}
	if node.Name != other.Name {
		panic("comparing mismatched nodes")
	}

	return node.Data.FileInfo.Compare(other.Data.FileInfo)
}