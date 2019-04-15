package filetree

import (
	"github.com/google/uuid"
	"strings"
)

const (
	newLine              = "\n"
	noBranchSpace        = "    "
	branchSpace          = "│   "
	middleItem           = "├─"
	lastItem             = "└─"
	whiteoutPrefix       = ".wh."
	doubleWhiteoutPrefix = ".wh..wh.."
	uncollapsedItem      = "─ "
	collapsedItem        = "⊕ "
)

// NewFileTree creates an empty FileTree
func NewFileTree() (tree *FileTree) {
	tree = new(FileTree)
	tree.Size = 0
	tree.Root = new(FileNode)
	tree.Root.Tree = tree
	tree.Root.Children = make(map[string]*FileNode)
	tree.Id = uuid.New()
	return tree
}

// AddPath 向树中添加具有给定负载的新节点
func (tree *FileTree) AddPath(path string, data FileInfo) (*FileNode, []*FileNode, error) {
	nodeName := strings.Split(strings.Trim(path, "/"), "/")
	node := tree.Root
	addedNodes := make([]*FileTree, 0)
	for idx, name := range nodeName{
		if name == "" {
			continue
		}

		if node.Children[name] != nil {
			node = node.Children[name]
		} else {
			node = node.AddChild(name, FileInfo{})
		}
	}
}