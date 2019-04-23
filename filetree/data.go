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

// NewNodeData 为FileNode创建空的 NodeData 结构
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

// Copy 复制文件信息
func (data *FileInfo) Copy() *FileInfo {
	if data == nil {
		return nil
	}
	return &FileInfo{
		Path:     data.Path,
		TypeFlag: data.TypeFlag,
		LinkName: data.LinkName,
		hash:     data.hash,
		Size:     data.Size,
		Mode:     data.Mode,
		Uid:      data.Uid,
		Gid:      data.Gid,
		IsDir:    data.IsDir,
	}
}

// merge 将两个DiffType合并为一个结果。本质上，返回给定值，除非两个值不同，在这种情况下，我们只能确定存在"change".
func (diff DiffType) merge(other DiffType) DiffType {
	if diff == other {
		return diff
	}
	return Changed
}