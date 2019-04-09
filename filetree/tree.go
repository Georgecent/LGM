package filetree

import "github.com/google/uuid"

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