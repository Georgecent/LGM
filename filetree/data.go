package filetree

import (
	"archive/tar"
	"github.com/cespare/xxhash"
	"github.com/sirupsen/logrus"
	"io"
)

const (
	Unchanged DiffType = iota
	Changed
	Added
	Removed
)

var GlobalFileTreeCollapse bool

// NewNodeData creates an empty NodeData struct for a FileNode
func NewNodeData() *NodeData {
	return &NodeData{
		ViewInfo: *NewViewInfo(),
		FileInfo: FileInfo{},
		DiffType: Unchanged,
	}
}

func NewViewInfo() (view *ViewInfo) {
	return &ViewInfo{
		Collapsed: 	GlobalFileTreeCollapse,
		Hidden:		false,
	}
}

func getHashFromReader(reader io.Reader) uint64 {
	h := xxhash.New()
	
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			logrus.Panic(err)
		}
		if n == 0 {
			break
		}

		h.Write(buf[:n])
	}
	// Sum64 returns the current hash.
	return h.Sum64()
}

// NewFileInfo从tar头和文件内容中提取元数据，并生成新的FileInfo对象。
func NewFileInfo(reader *tar.Reader, header *tar.Header, path string) FileInfo {
	if header.Typeflag == tar.TypeDir{
		return FileInfo{
			Path:		path,
			TypeFlag:	header.Typeflag,
			LinkName: 	header.Linkname,
			hash:		0,
			Size: 		header.FileInfo().Size(),
			Mode: 		header.FileInfo().Mode(),
			Uid: 		header.Uid,
			Gid: 		header.Gid,
			IsDir: 		header.FileInfo().IsDir(),
		}
	}

	hash := getHashFromReader(reader)

	return FileInfo{
		Path:     path,
		TypeFlag: header.Typeflag,
		LinkName: header.Linkname,
		hash:     hash,
		Size:     header.FileInfo().Size(),
		Mode:     header.FileInfo().Mode(),
		Uid:      header.Uid,
		Gid:      header.Gid,
		IsDir:    header.FileInfo().IsDir(),
	}
}
