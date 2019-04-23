package filetree

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sort"
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

// NewFileTree 创建一个空的FileTree
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
	for idx, name := range nodeNames {
		if name == "" {
			continue
		}
		// find or create node
		if node.Children[name] != nil {
			node = node.Children[name]
		} else {
			// 不要附加有效载荷。 有效负载的目的地是Path的终端节点，而不是任何中间节点。
			node = node.AddChild(name, FileInfo{})
			addedNodes = append(addedNodes, node)

			if node == nil {
				// the child could not be added
				return node, addedNodes, fmt.Errorf(fmt.Sprintf("could not add child node '%s'", name))
			}
		}

		// attach payload to the last specified node
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

// VisitDepthChildFirst 深度优先迭代给定的树，首先评估最深的深度（冒泡访问）
// 访问者模式
func (tree *FileTree) VisitDepthChildFirst(visitor Visitor, evaluator VisitEvaluator) error {
	return tree.Root.VisitDepthChildFirst(visitor, evaluator)
}

// VisitDepthParentFirst 深度优先迭代给定的树，首先评估最浅的深度（下沉时访问）
func (tree *FileTree) VisitDepthParentFirst(visitor Visitor, evaluator VisitEvaluator) error {
	return tree.Root.VisitDepthParentFirst(visitor, evaluator)
}

// StringBetween 以ASCII表示形式返回部分树。
func (tree *FileTree) StringBetween(start, stop int, showAttributes bool) string {
	return tree.renderStringTreeBetween(start, stop, showAttributes)
}

// renderParams是更大树的上下文中的FileNode的表示。 存储的所有数据对于以树格式呈现单行是必需的。
type renderParams struct {
	node          *FileNode
	spaces        []bool
	childSpaces   []bool
	showCollapsed bool
	isLast        bool
}

// renderStringTreeBetween 返回表示给定行之间给定树的字符串。 由于每个节点都在其自己的行上呈现，因此返回的字符串显示不受折叠父级影响的可见节点。
func (tree *FileTree) renderStringTreeBetween(startRow, stopRow int, showAttributes bool) string {
	// generate a list of nodes to render(生成要渲染的节点列表)
	var params = make([]renderParams, 0)
	var result string

	// visit from the front of the list
	var paramsToVisit = []renderParams{{node: tree.Root, spaces: []bool{}, showCollapsed: false, isLast: false}}
	for currentRow := 0; len(paramsToVisit) > 0 && currentRow <= stopRow; currentRow++ {
		// pop the first node
		var currentParams renderParams
		currentParams, paramsToVisit = paramsToVisit[0], paramsToVisit[1:]

		// 记下稍后要访问的下一个node
		var keys []string
		for key := range currentParams.node.Children {
			keys = append(keys, key)
		}
		// 按顺序访问nodes
		sort.Strings(keys)

		var childParams = make([]renderParams, 0)
		for idx, name := range keys {
			child := currentParams.node.Children[name]
			// don't visit this node...
			if child.Data.ViewInfo.Hidden || currentParams.node.Data.ViewInfo.Collapsed {
				continue
			}

			// visit this node...
			isLast := idx == (len(currentParams.node.Children) - 1)
			showCollapsed := child.Data.ViewInfo.Collapsed && len(child.Children) > 0

			// completely copy the reference slice
			childSpaces := make([]bool, len(currentParams.childSpaces))
			copy(childSpaces, currentParams.childSpaces)

			if len(child.Children) > 0 && !child.Data.ViewInfo.Collapsed {
				childSpaces = append(childSpaces, isLast)
			}

			childParams = append(childParams, renderParams{
				node:          child,
				spaces:        currentParams.childSpaces,
				childSpaces:   childSpaces,
				showCollapsed: showCollapsed,
				isLast:        isLast,
			})
		}
		// keep the child nodes to visit later
		paramsToVisit = append(childParams, paramsToVisit...)

		// never process the root node
		if currentParams.node == tree.Root {
			currentRow--
			continue
		}

		// process the current node
		if currentRow >= startRow && currentRow <= stopRow {
			params = append(params, currentParams)
		}
	}

	// render the result
	for idx := range params {
		currentParams := params[idx]

		if showAttributes {
			result += currentParams.node.MetadataString() + " "
		}
		result += currentParams.node.renderTreeLine(currentParams.spaces, currentParams.isLast, currentParams.showCollapsed)
	}

	return result
}

func (tree *FileTree) VisibleSize() int {
	var size int

	visitor := func(node *FileNode) error {
		size++
		return nil
	}
	visitEvaluator := func(node *FileNode) bool {
		return !node.Data.ViewInfo.Collapsed && !node.Data.ViewInfo.Hidden
	}
	err := tree.VisitDepthParentFirst(visitor, visitEvaluator)
	if err != nil {
		logrus.Errorf("unable to determine visible tree size: %+v", err)
	}

	return size
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

// RemovePath 在给定其路径的情况下从树中删除节点。
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

// CompareAndMark 与给定（上部）树进行比较时，使用DiffType注释标记拥有（下部）树中的FileNodes。
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

		// 该文件存在于较低layer
		lowerNode, _ := tree.GetNode(upperNode.Path())
		diffType := lowerNode.compare(upperNode)
		modifications = append(modifications, compareMark{lowerNode: lowerNode, upperNode: upperNode, tentative: diffType, final: -1})

		return nil
	}
	// 我们必须从叶子向上访问，以确保可以从子项中派生和分配差异类型
	err := upper.VisitDepthChildFirst(graft, nil)
	if err != nil {
		return err
	}

	// 注意所属树中每个注释的比较结果。
	for _, pair := range modifications {
		if pair.final > 0 {
			pair.lowerNode.AssignDiffType(pair.final)
		} else if pair.lowerNode.Data.DiffType == Unchanged {
			pair.lowerNode.deriveDiffType(pair.tentative)
		}

		// 在拥有的树上保持上层的有效负载
		pair.lowerNode.Data.FileInfo = *pair.upperNode.Data.FileInfo.Copy()
	}
	return nil
}

// markRemoved 将给定路径处的filenode注释为已删除。
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