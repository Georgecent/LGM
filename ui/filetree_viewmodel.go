package ui

import (
	"LGM/filetree"
	"bytes"
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
