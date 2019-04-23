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
	if parent != nil {
		node.Tree = parent.Tree
	}
	return node
}

// String shows the filename formatted into the proper color (by DiffType), additionally indicating if it is a symlink.
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

// renderTreeLine returns a string representing this FileNode in the context of a greater ASCII tree.
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

// AddChild creates a new node relative to the current FileNode.
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

// VisitDepthParentFirst iterates a tree depth-first (starting at this FileNode), evaluating the shallowest depths first (visit while sinking down)
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

// Compare determines the DiffType between two FileInfos based on the type and contents of each given FileInfo
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
			// don't include file sizes of children that have been removed (unless the node in question is a removed dir,
			// then show the accumulated size of removed files)
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