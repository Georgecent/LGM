package filetree

import (
	"github.com/google/uuid"
	"os"
)

// FileTree 表示一组文件、目录及其关系
type FileTree struct {
	Root 		*FileNode
	Size 		int
	FileSize 	uint64
	Name 		string
	Id 			uuid.UUID
}

// FileNode表示单个文件，它与下面文件的关系，它存在的树以及给定文件的元数据。
type FileNode struct {
	Tree 		*FileTree
	Parent		*FileNode
	Name 		string
	Data		NodeData
	Children	map[string]*FileNode
	path 		string
}

// NodeData是FileNode的有效负载
type NodeData struct {
	ViewInfo 	ViewInfo
	FileInfo 	FileInfo
	DiffType 	DiffType
}

// ViewInfo包含特定FileNode的UI特定详细信息
type ViewInfo struct {
	Collapsed 	bool
	Hidden		bool
}

// FileInfo包含特定FileNode的tar元数据
type FileInfo struct {
	Path 		string
	TypeFlag	byte
	LinkName	string
	hash 		uint64
	Size 		int64
	Mode 		os.FileMode
	Uid 		int
	Gid 		int
	IsDir 		bool
}

// DiffType定义两个FileNode之间的比较结果
type DiffType int

// EfficiencyData表示给定文件树路径的存储和引用统计信息。
type EfficiencyData struct {
	Path				string
	Nodes 				[]*FileNode
	CumulativeSize		int64
	minDiscoveredSize 	int64
}

// EfficiencySlice表示一组有序的EfficiencyData数据结构。
type EfficiencySlice []*EfficiencyData