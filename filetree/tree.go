package filetree

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	nodeNames := strings.Split(strings.Trim(path, "/"), "/")
	node := tree.Root
	addedNodes := make([]*FileNode, 0)
	for idx, name := range nodeNames{
		if name == "" {
			continue
		}

		if node.Children[name] != nil {
			node = node.Children[name]
		} else {
			node = node.AddChild(name, FileInfo{})
			addedNodes = append(addedNodes, node)

			if node == nil {
				// the child could not be added
				return node, addedNodes, fmt.Errorf(fmt.Sprintf("could not add child node '%s'", name))
			}
		}

		if idx == len(nodeNames)-1 {
			node.Data.FileInfo = data
		}
	}

	return node, addedNodes, nil
}

// Visitor 处理、观察或以其他方式转换给定节点
type Visitor func(*FileNode) error

// VisitEvaluator 用于指示访问者是否应该访问给定的节点
type VisitEvaluator func(*FileNode) bool

// VisitDepthChildFirst iterates the given tree depth-first, evaluating the deepest depths first (visit on bubble up)
// 访问者模式
func (tree *FileTree) VisitDepthChildFirst(visitor Visitor, evaluator VisitEvaluator) error {
	return tree.Root.VisitDepthChildFirst(visitor, evaluator)
}

// Copy 返回给定文件树的副本
func (tree *FileTree) Copy() *FileTree {
	newTree := NewFileTree()
	newTree.Size = tree.Size
	newTree.FileSize = tree.FileSize
	newTree.Root = tree.Root.Copy(newTree.Root)

	// update the tree pointers
	err := newTree.VisitDepthChildFirst(func(node *FileNode) error{
		node.Tree = newTree
		return nil
	}, nil)
}

// StackTreeRange 将一系列树组合成一棵树
func StackTreeRange(trees []*FileTree, start, stop int) *FileTree {
	tree := trees[0].Copy()
	for idx := start; idx <= stop; idx++ {
		err := tree.Stack(trees[idx])
		if err != nil {
			logrus.Errorf("could not stack tree range: %v", err)
		}
	}
	return tree
}