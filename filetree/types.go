package filetree

import "github.com/google/uuid"

// FileTree 表示一组文件、目录及其关系
type FileTree struct {
	Root 		*FileNode
	Size 		int
	FileSize 	uint64
	Name 		string
	Id 			uuid.UUID
}

type FileNode struct {
	Tree 		*FileTree
	Parent		*FileNode
	Name 		string
	Data		NodeData
	Children	map[string]*FileNode
	path 		string
}

type NodeData struct {

}