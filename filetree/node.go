package filetree

import (
	"archive/tar"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/phayes/permbits"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
)

const (
	AttributeFormat = "%s%s %11s %10s "
)

var diffTypeColor = map[DiffType]*color.Color{
	Added:     color.New(color.FgGreen),
	Removed:   color.New(color.FgRed),
	Changed:   color.New(color.FgYellow),
	Unchanged: color.New(color.Reset),
}


// IsWhiteout 返回此文件是否可能是overlay-whiteout文件。
func (node *FileNode) IsWhiteout() bool {
	return strings.HasPrefix(node.Name, whiteoutPrefix)
}

// NewNode 使用有效负载创建相对于给定父节点的新FileNode。
func NewNode(parent *FileNode, name string, data FileInfo) (node *FileNode)  {
	node = new(FileNode)
	node.Name = name
	node.Data = *NewNodeData()
	node.Data.FileInfo = *data.Copy()

	node.Children = make(map[string]*FileNode)
	node.Parent = parent
	if parent != nil {
		node.Tree = parent.Tree
	}
	return node
}

// String 显示格式化为正确颜色的文件名（通过DiffType），另外指示它是否是符号链接。
func (node *FileNode) String() string {
	var display string
	if node == nil {
		return ""
	}

	display = node.Name
	if node.Data.FileInfo.TypeFlag == tar.TypeSymlink || node.Data.FileInfo.TypeFlag == tar.TypeLink {
		display += " → " + node.Data.FileInfo.LinkName
	}
	return diffTypeColor[node.Data.DiffType].Sprint(display)
}

// renderTreeLine 在更大的ASCII树的上下文中返回表示此FileNode的字符串。
func (node *FileNode) renderTreeLine(spaces []bool, last bool, collapsed bool) string {
	var otherBranches string
	for _, space := range spaces {
		if space {
			otherBranches += noBranchSpace
		} else {
			otherBranches += branchSpace
		}
	}

	thisBranch := middleItem
	if last {
		thisBranch = lastItem
	}

	collapsedIndicator := uncollapsedItem
	if collapsed {
		collapsedIndicator = collapsedItem
	}

	return otherBranches + thisBranch + collapsedIndicator + node.String() + newLine
}

// AddChild 创建一个相对于当前FileNode的新节点。
func (node *FileNode) AddChild(name string, data FileInfo) (child *FileNode)  {
	// never allow processing of purely whiteout flag files (for now)
	if strings.HasPrefix(name, doubleWhiteoutPrefix) {
		return nil
	}

	child = NewNode(node, name, data)
	if node.Children[name] != nil {
		// tree node already exists, replace the payload, keep the children
		node.Children[name].Data.FileInfo = *data.Copy()
	} else {
		//fmt.Printf("In AddChild TreeSize: %d\n", node.Tree.Size)
		node.Children[name] = child
		node.Tree.Size++
	}

	return child
}

// Copy 相对于新父节点复制现有节点。
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

// VisitDepthChildFirst 深度优先迭代树（从此FileNode开始），首先评估最深的深度（冒泡访问）
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

// VisitDepthParentFirst 深度优先迭代树（从此FileNode开始），首先评估最浅的深度（下沉时访问）
func (node *FileNode) VisitDepthParentFirst(visitor Visitor, evaluator VisitEvaluator) error {
	var err error

	doVisit := evaluator != nil && evaluator(node) || evaluator == nil

	if !doVisit {
		return nil
	}

	// never visit the root node
	if node != node.Tree.Root {
		err = visitor(node)
		if err != nil {
			return err
		}
	}

	var keys []string
	for key := range node.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, name := range keys {
		child := node.Children[name]
		err = child.VisitDepthParentFirst(visitor, evaluator)
		if err != nil {
			return err
		}
	}
	return err
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

// Remove 从它的父FileNode的关系中删除当前的FileNode。
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

// Compare 根据每个给定FileInfo的类型和内容确定两个FileInfos之间的DiffType
func (data *FileInfo) Compare(other FileInfo) DiffType {
	if data.TypeFlag == other.TypeFlag {
		if data.hash == other.hash &&
			data.Mode == other.Mode &&
			data.Uid == other.Uid &&
			data.Gid == other.Gid {
			return Unchanged
		}
	}
	return Changed
}

// compare 针对给定节点的当前节点，返回确定的DiffType。
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

// MetadatString 以列式字符串形式返回FileNode元数据。
func (node *FileNode) MetadataString() string {
	if node == nil {
		return ""
	}

	fileMode := permbits.FileMode(node.Data.FileInfo.Mode).String()
	dir := "-"
	if node.Data.FileInfo.IsDir {
		dir = "d"
	}
	user := node.Data.FileInfo.Uid
	group := node.Data.FileInfo.Gid
	userGroup := fmt.Sprintf("%d:%d", user, group)

	var sizeBytes int64

	if node.IsLeaf() {
		sizeBytes = node.Data.FileInfo.Size
	} else {
		sizer := func(curNode *FileNode) error {
			// 不包括已删除的子项的文件大小（除非有问题的节点是已删除的目录，然后显示已删除文件的累计大小）
			if curNode.Data.DiffType != Removed || node.Data.DiffType == Removed {
				sizeBytes += curNode.Data.FileInfo.Size
			}
			return nil
		}

		err := node.VisitDepthChildFirst(sizer, nil)
		if err != nil {
			logrus.Errorf("unable to propagate node for metadata: %+v", err)
		}
	}

	size := humanize.Bytes(uint64(sizeBytes))

	return diffTypeColor[node.Data.DiffType].Sprint(fmt.Sprintf(AttributeFormat, dir, fileMode, userGroup, size))
}