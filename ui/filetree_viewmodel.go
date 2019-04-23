package ui

import (
	"LGM/filetree"
	"LGM/utils"
	"bytes"
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"regexp"
	"strings"
)

// FileTreeViewModel 保存用于填充右窗格的UI对象和数据模型。 特别是显示所选图层或聚合文件ASCII树的窗格。
type FileTreeViewModel struct {
	ModelTree *filetree.FileTree
	ViewTree  *filetree.FileTree
	RefTrees  []*filetree.FileTree
	cache     filetree.TreeCache

	CollapseAll           bool
	ShowAttributes        bool
	HiddenDiffTypes       []bool
	TreeIndex             int
	bufferIndex           int
	bufferIndexLowerBound int

	refHeight int
	refWidth  int

	mainBuf bytes.Buffer
}

func NewFileTreeViewModel(tree *filetree.FileTree, refTrees []*filetree.FileTree, cache filetree.TreeCache) (treeViewModel *FileTreeViewModel) {
	treeViewModel = new(FileTreeViewModel)

	// populate main fields
	treeViewModel.ShowAttributes = viper.GetBool("filetree.show-attributes")
	treeViewModel.CollapseAll = viper.GetBool("filetree.collapse-dir")
	treeViewModel.ModelTree = tree
	treeViewModel.RefTrees = refTrees
	treeViewModel.cache = cache
	treeViewModel.HiddenDiffTypes = make([]bool, 4)

	hiddenTypes := viper.GetStringSlice("diff.hide")
	for _, hType := range hiddenTypes {
		switch t := strings.ToLower(hType); t {
		case "added":
			treeViewModel.HiddenDiffTypes[filetree.Added] = true
		case "removed":
			treeViewModel.HiddenDiffTypes[filetree.Removed] = true
		case "changed":
			treeViewModel.HiddenDiffTypes[filetree.Changed] = true
		case "unchanged":
			treeViewModel.HiddenDiffTypes[filetree.Unchanged] = true
		default:
			utils.PrintAndExit(fmt.Sprintf("unknown diff.hide value: %s", t))
		}
	}

	return treeViewModel
}

// Setup 在全局[gocui]视图对象的上下文中初始化UI关注点。
func (vm *FileTreeViewModel) Setup(lowerBound, height int) {
	vm.bufferIndexLowerBound = lowerBound
	vm.refHeight = height
}

// height 返回当前高度并考虑标题
func (vm *FileTreeViewModel) height() int {
	if vm.ShowAttributes {
		return vm.refHeight - 1
	}
	return vm.refHeight
}

// bufferIndexUpperBound 返回视图的当前上限
func (vm *FileTreeViewModel) bufferIndexUpperBound() int {
	return vm.bufferIndexLowerBound + vm.height()
}

// IsVisible 指示文件树视图窗格当前是否已初始化
func (vm *FileTreeViewModel) IsVisible() bool {
	if vm == nil {
		return false
	}
	return true
}

// resetCursor 将光标移回缓冲区的顶部并转换为缓冲区的顶部。
func (vm *FileTreeViewModel) resetCursor() {
	vm.TreeIndex = 0
	vm.bufferIndex = 0
	vm.bufferIndexLowerBound = 0
}

// setTreeByLayer 通过堆叠指示的图像层文件树来填充视图模型。
func (vm *FileTreeViewModel) setTreeByLayer(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop int) error {
	if topTreeStop > len(vm.RefTrees)-1 {
		return fmt.Errorf("invalid layer index given: %d of %d", topTreeStop, len(vm.RefTrees)-1)
	}
	newTree := vm.cache.Get(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop)

	// preserve vm state on copy
	visitor := func(node *filetree.FileNode) error {
		newNode, err := newTree.GetNode(node.Path())
		if err == nil {
			newNode.Data.ViewInfo = node.Data.ViewInfo
		}
		return nil
	}
	err := vm.ModelTree.VisitDepthChildFirst(visitor, nil)
	if err != nil {
		logrus.Errorf("unable to propagate layer tree: %+v", err)
		return err
	}

	vm.ModelTree = newTree
	return nil
}

// doCursorUp 在光标上执行内部视图的缓冲区调整。 注意：这与gocui缓冲区无关。
func (vm *FileTreeViewModel) CursorUp() bool {
	if vm.TreeIndex <= 0 {
		return false
	}
	vm.TreeIndex--
	if vm.TreeIndex < vm.bufferIndexLowerBound {
		vm.bufferIndexLowerBound--
	}
	if vm.bufferIndex > 0 {
		vm.bufferIndex--
	}
	return true
}

// doCursorDown 在光标向下执行内部视图的缓冲区调整。 注意：这与gocui缓冲区无关。
func (vm *FileTreeViewModel) CursorDown() bool {
	if vm.TreeIndex >= vm.ModelTree.VisibleSize() {
		return false
	}
	vm.TreeIndex++
	if vm.TreeIndex > vm.bufferIndexUpperBound() {
		vm.bufferIndexLowerBound++
	}
	vm.bufferIndex++
	if vm.bufferIndex > vm.height() {
		vm.bufferIndex = vm.height()
	}
	return true
}

// CursorLeft 将光标向上移动，直到我们到达父节点或树的顶部
func (vm *FileTreeViewModel) CursorLeft(filterRegex *regexp.Regexp) error {
	var visitor func(*filetree.FileNode) error
	var evaluator func(*filetree.FileNode) bool
	var dfsCounter, newIndex int
	oldIndex := vm.TreeIndex
	currentNode := vm.getAbsPositionNode(filterRegex)

	if currentNode == nil {
		return nil
	}
	parentPath := currentNode.Parent.Path()

	visitor = func(curNode *filetree.FileNode) error {
		if strings.Compare(parentPath, curNode.Path()) == 0 {
			newIndex = dfsCounter
		}
		dfsCounter++
		return nil
	}

	evaluator = func(curNode *filetree.FileNode) bool {
		regexMatch := true
		if filterRegex != nil {
			match := filterRegex.Find([]byte(curNode.Path()))
			regexMatch = match != nil
		}
		return !curNode.Parent.Data.ViewInfo.Collapsed && !curNode.Data.ViewInfo.Hidden && regexMatch
	}

	err := vm.ModelTree.VisitDepthParentFirst(visitor, evaluator)
	if err != nil {
		logrus.Errorf("could not propagate tree on cursorLeft: %+v", err)
		return err
	}

	vm.TreeIndex = newIndex
	moveIndex := oldIndex - newIndex
	if newIndex < vm.bufferIndexLowerBound {
		vm.bufferIndexLowerBound = vm.TreeIndex
	}

	if vm.bufferIndex > moveIndex {
		vm.bufferIndex = vm.bufferIndex - moveIndex
	} else {
		vm.bufferIndex = 0
	}

	return nil
}

// CursorRight 如果需要，可以进入扩展目录的目录
func (vm *FileTreeViewModel) CursorRight(filterRegex *regexp.Regexp) error {
	node := vm.getAbsPositionNode(filterRegex)
	if node == nil {
		return nil
	}

	if !node.Data.FileInfo.IsDir {
		return nil
	}

	if len(node.Children) == 0 {
		return nil
	}

	if node.Data.ViewInfo.Collapsed {
		node.Data.ViewInfo.Collapsed = false
	}

	vm.TreeIndex++
	if vm.TreeIndex > vm.bufferIndexUpperBound() {
		vm.bufferIndexLowerBound++
	}

	vm.bufferIndex++
	if vm.bufferIndex > vm.height() {
		vm.bufferIndex = vm.height()
	}

	return nil
}

// PageDown 移动到下一页，将光标置于顶部
func (vm *FileTreeViewModel) PageDown() error {
	nextBufferIndexLowerBound := vm.bufferIndexLowerBound + vm.height()
	nextBufferIndexUpperBound := nextBufferIndexLowerBound + vm.height()

	// todo: this work should be saved or passed to render...
	treeString := vm.ViewTree.StringBetween(nextBufferIndexLowerBound, nextBufferIndexUpperBound, vm.ShowAttributes)
	lines := strings.Split(treeString, "\n")

	newLines := len(lines) - 1
	if vm.height() >= newLines {
		nextBufferIndexLowerBound = vm.bufferIndexLowerBound + newLines
	}

	vm.bufferIndexLowerBound = nextBufferIndexLowerBound

	if vm.TreeIndex < nextBufferIndexLowerBound {
		vm.bufferIndex = 0
		vm.TreeIndex = nextBufferIndexLowerBound
	} else {
		vm.bufferIndex = vm.bufferIndex - newLines
	}

	return nil
}

// PageUp 移动到上一页，将光标置于顶部
func (vm *FileTreeViewModel) PageUp() error {
	nextBufferIndexLowerBound := vm.bufferIndexLowerBound - vm.height()
	nextBufferIndexUpperBound := nextBufferIndexLowerBound + vm.height()

	// todo: this work should be saved or passed to render...
	treeString := vm.ViewTree.StringBetween(nextBufferIndexLowerBound, nextBufferIndexUpperBound, vm.ShowAttributes)
	lines := strings.Split(treeString, "\n")

	newLines := len(lines) - 2
	if vm.height() >= newLines {
		nextBufferIndexLowerBound = vm.bufferIndexLowerBound - newLines
	}

	vm.bufferIndexLowerBound = nextBufferIndexLowerBound

	if vm.TreeIndex > (nextBufferIndexUpperBound - 1) {
		vm.bufferIndex = 0
		vm.TreeIndex = nextBufferIndexLowerBound
	} else {
		vm.bufferIndex = vm.bufferIndex + newLines
	}
	return nil
}

// getAbsPositionNode 确定所选屏幕光标在文件树中的位置，返回所选的FileNode。
func (vm *FileTreeViewModel) getAbsPositionNode(filterRegex *regexp.Regexp) (node *filetree.FileNode) {
	var visitor func(*filetree.FileNode) error
	var evaluator func(*filetree.FileNode) bool
	var dfsCounter int

	visitor = func(curNode *filetree.FileNode) error {
		if dfsCounter == vm.TreeIndex {
			node = curNode
		}
		dfsCounter++
		return nil
	}

	evaluator = func(curNode *filetree.FileNode) bool {
		regexMatch := true
		if filterRegex != nil {
			match := filterRegex.Find([]byte(curNode.Path()))
			regexMatch = match != nil
		}
		return !curNode.Parent.Data.ViewInfo.Collapsed && !curNode.Data.ViewInfo.Hidden && regexMatch
	}

	err := vm.ModelTree.VisitDepthParentFirst(visitor, evaluator)
	if err != nil {
		logrus.Errorf("unable to get node position: %+v", err)
	}

	return node
}

// toggleCollapse 将折叠/展开选定的FileNode。
func (vm *FileTreeViewModel) toggleCollapse(filterRegex *regexp.Regexp) error {
	node := vm.getAbsPositionNode(filterRegex)
	if node != nil && node.Data.FileInfo.IsDir {
		node.Data.ViewInfo.Collapsed = !node.Data.ViewInfo.Collapsed
	}
	return nil
}

// toggleCollapseAll 将折叠/展开所有目录。
func (vm *FileTreeViewModel) toggleCollapseAll() error {
	vm.CollapseAll = !vm.CollapseAll

	visitor := func(curNode *filetree.FileNode) error {
		curNode.Data.ViewInfo.Collapsed = vm.CollapseAll
		return nil
	}

	evaluator := func(curNode *filetree.FileNode) bool {
		return curNode.Data.FileInfo.IsDir
	}

	err := vm.ModelTree.VisitDepthChildFirst(visitor, evaluator)
	if err != nil {
		logrus.Errorf("unable to propagate tree on toggleCollapseAll: %+v", err)
	}

	return nil
}

// toggleCollapse 将折叠/展开选定的FileNode。
func (vm *FileTreeViewModel) toggleAttributes() error {
	vm.ShowAttributes = !vm.ShowAttributes
	return nil
}

// toggleShowDiffType 将在filetree窗格中显示/隐藏选定的DiffType。
func (vm *FileTreeViewModel) toggleShowDiffType(diffType filetree.DiffType) error {
	vm.HiddenDiffTypes[diffType] = !vm.HiddenDiffTypes[diffType]

	return nil
}

// Update 刷新状态对象以供将来呈现。
func (vm *FileTreeViewModel) Update(filterRegex *regexp.Regexp, width, height int) error {
	vm.refWidth = width
	vm.refHeight = height

	// keep the vm selection in parity with the current DiffType selection
	err := vm.ModelTree.VisitDepthChildFirst(func(node *filetree.FileNode) error {
		node.Data.ViewInfo.Hidden = vm.HiddenDiffTypes[node.Data.DiffType]
		visibleChild := false
		for _, child := range node.Children {
			if !child.Data.ViewInfo.Hidden {
				visibleChild = true
				node.Data.ViewInfo.Hidden = false
			}
		}
		// hide nodes that do not match the current file filter regex (also don't unhide nodes that are already hidden)
		if filterRegex != nil && !visibleChild && !node.Data.ViewInfo.Hidden {
			match := filterRegex.FindString(node.Path())
			node.Data.ViewInfo.Hidden = len(match) == 0
		}
		return nil
	}, nil)

	if err != nil {
		logrus.Errorf("unable to propagate vm model tree: %+v", err)
		return err
	}

	// make a new tree with only visible nodes
	vm.ViewTree = vm.ModelTree.Copy()
	err = vm.ViewTree.VisitDepthParentFirst(func(node *filetree.FileNode) error {
		if node.Data.ViewInfo.Hidden {
			vm.ViewTree.RemovePath(node.Path())
		}
		return nil
	}, nil)

	if err != nil {
		logrus.Errorf("unable to propagate vm view tree: %+v", err)
		return err
	}

	return nil
}

// Render 将状态对象（文件树）刷新到窗格。
func (vm *FileTreeViewModel) Render() error {
	treeString := vm.ViewTree.StringBetween(vm.bufferIndexLowerBound, vm.bufferIndexUpperBound(), vm.ShowAttributes)
	lines := strings.Split(treeString, "\n")

	// update the contents
	vm.mainBuf.Reset()
	for idx, line := range lines {
		if idx == vm.bufferIndex {
			fmt.Fprintln(&vm.mainBuf, Formatting.Selected(vtclean.Clean(line, false)))
		} else {
			fmt.Fprintln(&vm.mainBuf, line)
		}
	}
	return nil
}
