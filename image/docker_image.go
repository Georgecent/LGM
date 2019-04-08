package image

import (
	"LGM/filetree"
	"io"
)

func newDockerImageAnalyzer(imageId string) Analyzer {
	return &dockerImageAnalyzer{
		// store discovered json files in a map so we can read the image in one pass
		jsonFiles: make(map[string][]byte),
		layerMap:  make(map[string]*filetree.FileTree),
		id:        imageId,
	}
}

func (image *dockerImageAnalyzer) Fetch() (io.ReadCloser, error) {

}

func (image *dockerImageAnalyzer) Parse(tarFile io.ReadCloser) error{

}

func (image *dockerImageAnalyzer) Analyze() (*AnalysisResult, error){

}