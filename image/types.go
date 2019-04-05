package image

import (
	"LGM/filetree"
	"github.com/docker/docker/client"
	"io"
)

type Parser interface {
}

type Analyzer interface {
	Fetch() (io.ReadCloser, error)
	Parse(io.ReadCloser) error
	Analyze() (*AnalysisResult, error)
}

type Layer interface {
	Id() string
	ShortId() string
	Index() int
	Command() string
	Size() uint64
	Tree() *filetree.FileTree
	String() string
}

type AnalysisResult struct {
	Layers            []Layer
	RefTrees          []*filetree.FileTree
	Efficiency        float64
	SizeBytes         uint64
	UserSizeByes      uint64  // 这是除基本图像之外的所占字节
	WastedUserPercent float64 // WastedUserPercent = wasted-bytes/user-size-bytes
	WastedBytes       uint64
	Inefficiencies    filetree.EfficiencySlice
}

type dockerImageAnalyzer struct {
	id        string
	client    *client.Client
	jsonFiles map[string][]byte
	trees     []*filetree.FileTree
	layerMap  map[string]*filetree.FileTree
	layers    []*dockerLayer
}

// dockerImageHistoryEntry 表示Docker镜像历史记录条目
type dockerImageHistoryEntry struct {
	ID         string
	Size       uint64
	Created    string `json:"created"`
	Author     string `json:"author"`
	CreatedBy  string `json:"created_by"`
	EmptyLayer bool   `json:"empty_layer"`
}

type dockerImageManifest struct {
	ConfigPath    string   `json:"Config"`
	RepoTags      []string `json:"RepoTags"`
	LayerTarPaths []string `json:"Layers"`
}

type dockerRootFs struct {
	Type    string   `json:"type"`
	DiffIds []string `json:"diff_ids"`
}

type dockerImageConfig struct {
	History []dockerImageHistoryEntry `json:"history"`
	RootFs  dockerRootFs              `json:"rootfs"`
}

// dockerLayer 表示Docker镜像层和元数据
type dockerLayer struct {
	tarPath string
	history dockerImageHistoryEntry
	index   int
	tree    *filetree.FileTree
}