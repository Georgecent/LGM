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

	if err != nil {
		logrus.Errorf("unable to propagate tree on copy(): %+v", err)
	}

	return newTree
}

// Stack 将两棵树合并在一起。这是通过将给定的树“堆叠”到所属树的顶部来完成的。
func (tree *FileTree) Stack(upper *FileTree) error {
	graft := func(node *FileNode) error{
		if node.IsWhiteout() {
			err := tree.RemovePath(node.Path())
			if err != nil {
				return fmt.Errorf("cannot remove node %s: %v", node.Path(), err.Error())
			}
		} else {
			newNode, _, err := tree.AddPath(node.Path(), node.Data.FileInfo)
			if err != nil {
				return fmt.Errorf("cannot add node %s: %v", newNode.Path(), err.Error())
			}
		}
		return nil
	}
	return upper.VisitDepthChildFirst(graft, nil)
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

// RemovePath removes a node from the tree given its path.
func (tree *FileTree) RemovePath(path string) error {
	node, err := tree.GetNode(path)
	if err != nil {
		return err
	}
	return node.Remove()
}

// GetNode fetches a single node when given a slash-delimited string from root ('/') to the desired node (e.g. '/a/node/path')
func (tree *FileTree) GetNode(path string) (*FileNode, error) {
	nodeNames := strings.Split(strings.Trim(path, "/"), "/")
	node := tree.Root
	for _, name := range nodeNames {
		if name == "" {
			continue
		}
		if node.Children[name] == nil {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		node = node.Children[name]
	}
	return node, nil
}