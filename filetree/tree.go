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

// IsLeaf returns true is the current node has no child nodes.
func (node *FileNode) IsLeaf() bool {
	return len(node.Children) == 0
}

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

type compareMark struct {
	lowerNode *FileNode
	upperNode *FileNode
	// 试验
	tentative DiffType
	final     DiffType
}

// CompareAndMark marks the FileNodes in the owning (lower) tree with DiffType annotations when compared to the given (upper) tree.
func (tree *FileTree) CompareAndMark(upper *FileTree) error {
	// 总是比较原始的，未改变的树。
	originalTree := tree

	modifications := make([]compareMark, 0)

	graft := func(upperNode *FileNode) error{
		if upperNode.IsWhiteout() {
			err := tree.markRemoved(upperNode.Path())
			if err != nil {
				return fmt.Errorf("cannot remove upperNode %s: %v", upperNode.Path(), err.Error())
			}
			return nil
		}

		// 注意：由于我们没有与原始树进行比较（复制树很昂贵），我们可能会错误地将添加节点的父节点标记为已修改。 这将在以后更正。
		originalLowerNode, _ := originalTree.GetNode(upperNode.Path())

		if originalLowerNode == nil {
			_, newNodes, err := tree.AddPath(upperNode.Path(), upperNode.Data.FileInfo)
			if err != nil {
				return fmt.Errorf("cannot add new upperNode %s: %v", upperNode.Path(), err.Error())
			}
			for idx := len(newNodes) - 1; idx >= 0; idx-- {
				newNode := newNodes[idx]
				modifications = append(modifications, compareMark{lowerNode: newNode, upperNode: upperNode, tentative: -1, final: Added})
			}
			return nil
		}

		// the file exists in the lower layer
		lowerNode, _ := tree.GetNode(upperNode.Path())
		diffType := lowerNode.compare(upperNode)
		modifications = append(modifications, compareMark{lowerNode: lowerNode, upperNode: upperNode, tentative: diffType, final: -1})

		return nil
	}
	// we must visit from the leaves upwards to ensure that diff types can be derived from and assigned to children
	err := upper.VisitDepthChildFirst(graft, nil)
	if err != nil {
		return err
	}

	// take note of the comparison results on each note in the owning tree.
	for _, pair := range modifications {
		if pair.final > 0 {
			pair.lowerNode.AssignDiffType(pair.final)
		} else if pair.lowerNode.Data.DiffType == Unchanged {
			pair.lowerNode.deriveDiffType(pair.tentative)
		}

		// persist the upper's payload on the owning tree
		pair.lowerNode.Data.FileInfo = *pair.upperNode.Data.FileInfo.Copy()
	}
	return nil
}

// markRemoved annotates the FileNode at the given path as Removed.
func (tree *FileTree) markRemoved(path string) error {
	node, err := tree.GetNode(path)
	if err != nil {
		return err
	}
	return node.AssignDiffType(Removed)
}

// deriveDiffType 确定当前FileNode的DiffType。 注意：节点的DiffType始终是其属性及其内容的DiffType。 内容是目录子项的文件的字节。
func (node *FileNode) deriveDiffType(diffType DiffType) error{
	if node.IsLeaf() {
		return node.AssignDiffType(diffType)
	}

	myDiffType := diffType
	for _, v := range node.Children {
		myDiffType = myDiffType.merge(v.Data.DiffType)
	}

	return node.AssignDiffType(myDiffType)
}

// AssignDiffType 会将给定的DiffType分配给此节点，可能会影响子节点。
func (node *FileNode) AssignDiffType(diffType DiffType) error {
	var err error

	node.Data.DiffType = diffType

	if diffType == Removed {
		// if we've removed this node, then all children have been removed as well
		for _, child := range node.Children {
			err = child.AssignDiffType(diffType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}