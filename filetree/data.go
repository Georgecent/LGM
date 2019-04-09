package filetree

import "archive/tar"

func NewFileInfo(reader *tar.Reader, header *tar.Header, path string) FileInfo {
	if header.Typeflag {

	}
}
