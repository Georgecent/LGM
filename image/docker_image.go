package image

import (
	"LGM/filetree"
	"LGM/utils"
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var dockerVersion string

func newDockerImageAnalyzer(imageId string) Analyzer {
	return &dockerImageAnalyzer{
		// store discovered json files in a map so we can read the image in one pass
		jsonFiles: make(map[string][]byte),
		layerMap:  make(map[string]*filetree.FileTree),
		id:        imageId,
	}
}

func newDockerImageManifest(manifestBytes []byte) dockerImageManifest {
	var manifest []dockerImageManifest
	err := json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		logrus.Panic(err)
	}
	return manifest[0]
}

func newDockerImageConfig(configBytes []byte) dockerImageConfig{
	var imageConfig dockerImageConfig
	err := json.Unmarshal(configBytes, &imageConfig)
	if err != nil {
		logrus.Panic(err)
	}

	layerIdx := 0
	for idx := range imageConfig.History{
		if imageConfig.History[idx].EmptyLayer {
			imageConfig.History[idx].ID = "<missing>"
		} else {
			imageConfig.History[idx].ID = imageConfig.RootFs.DiffIds[layerIdx]
			layerIdx++
		}
	}
	return imageConfig
}

func (image *dockerImageAnalyzer) Fetch() (io.ReadCloser, error) {
	var err error

	// pull the image if it does not exist
	ctx := context.Background()

	host := os.Getenv("DOCKER_HOST")
	var clientOpts []func(*client.Client) error

	switch strings.Split(host, ":")[0] {
	case "ssh":
		// GetConnectionHelper returns Docker-specific connection helper for the given URL.
		// GetConnectionHelper returns nil without error when no helper is registered for the scheme.
		//
		// func GetConnectionHelper(daemonURL string) (*ConnectionHelper, error)
		// type ConnectionHelper struct {
		//    Dialer func(ctx context.Context, network, addr string) (net.Conn, error)
		//    Host   string // dummy URL used for HTTP requests. e.g. "http://docker"
		//	}
		//
		// ConnectionHelper allows to connect to a remote host with custom stream provider binary.
		helper, err := connhelper.GetConnectionHelper(host)
		if err != nil {
			fmt.Println("docker host", err)
		}
		clientOpts = append(clientOpts, func(c *client.Client) error {
			httpClient := &http.Client{
				Transport: &http.Transport{
					// https://www.godoc.org/net#Dialer
					// A Dialer contains options for connecting to an address.
					DialContext: helper.Dialer,
				},
			}
			// WithHTTPClient overrides the client http client with the specified one
			// func WithHTTPClient(client *http.Client) func(*Client) error
			return client.WithHTTPClient(httpClient)(c)
		})
		// WithHost overrides the client host with the specified one.
		clientOpts = append(clientOpts, client.WithHost(helper.Host))
		// WithDialContext applies the dialer to the client transport. This can be used to set the Timeout and KeepAlive settings of the client.
		clientOpts = append(clientOpts, client.WithDialContext(helper.Dialer))

	default:
		clientOpts = append(clientOpts, client.FromEnv)
	}

	// Todo init_version
	clientOpts = append(clientOpts, client.WithVersion(dockerVersion))

	// NewClientWithOpts initializes a new API client with default values. It takes functors to modify values when creating it, like `NewClientWithOpts(WithVersion(…))`
	// It also initializes the custom http headers to add to each request.
	//It won't send any version information if the version number is empty. It is highly recommended that you set a version or your client may break if the server is upgraded.
	image.client, err = client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return nil, err
	}

	// ImageInspectWithRaw returns the image information and its raw representation.
	_, _, err = image.client.ImageInspectWithRaw(ctx, image.id)
	if err != nil {
		// don't use the API, the CLI has more informative output
		fmt.Println("Image not available locally. Trying to pull '" + image.id + "'...")
		utils.RunDockerCmd("pull", image.id)
	}

	readCloser, err := image.client.ImageSave(ctx, []string{image.id})
	if err != nil {
		return nil, err
	}

	return readCloser, nil
}

func (image *dockerImageAnalyzer) Parse(tarFile io.ReadCloser) error{
	tarReader := tar.NewReader(tarFile)

	var currentLayer uint
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
			utils.Exit(1)
		}

		name := header.Name
		// Type '0' indicates a regular file.
		// TypeReg  = '0'
		// TypeSymlink = '2' //符号链接

		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeReg {
			// HasSuffix测试字符串是否以layer.tar后缀结尾。
			if strings.HasSuffix(name, "layer.tar") {
				currentLayer++
				if err != nil {
					return err
				}
				layerReader := tar.NewReader(tarReader)
				err := image.processLayerTar(name, currentLayer, layerReader)
				if err != nil {
					return err
				}
			} else if strings.HasSuffix(name, "json") {	//HasSuffix测试字符串是否以json后缀结尾。
				fileBuffer, err := ioutil.ReadAll(tarReader)
				if err != nil {
					return err
				}
				image.jsonFiles[name] = fileBuffer
			}
		}
	}

	return nil
}

func (image *dockerImageAnalyzer) Analyze() (*AnalysisResult, error){
	image.trees = make([]*filetree.FileTree, 0)

	manifest := newDockerImageManifest(image.jsonFiles["manifest.json"])
	config := newDockerImageConfig(image.jsonFiles[manifest.ConfigPath])

	// build the content tree
	for _, treeName := range manifest.LayerTarPaths {
		image.trees = append(image.trees, image.layerMap[treeName])
	}

	// build the layers array
	image.layers = make([]*dockerLayer, len(image.trees))

	// 请注意，图像配置以反向时间顺序存储图像，因此当按时间顺序迭代历史记录时，向后遍历图层（忽略没有图层内容的历史记录项）
	// Note: history is not required metadata in a docker image!
	tarPathIdx := 0
	histIdx := 0
	for layerIdx := len(image.trees) - 1; layerIdx >= 0; layerIdx-- {

		tree := image.trees[(len(image.trees)-1)-layerIdx]

		// ignore empty layers, we are only observing layers with content
		historyObj := dockerImageHistoryEntry{
			CreatedBy: "(missing)",
		}
		for nextHistIdx := histIdx; nextHistIdx < len(config.History); nextHistIdx++ {
			if !config.History[nextHistIdx].EmptyLayer {
				histIdx = nextHistIdx
				break
			}
		}
		if histIdx < len(config.History) && !config.History[histIdx].EmptyLayer {
			historyObj = config.History[histIdx]
			histIdx++
		}

		image.layers[layerIdx] = &dockerLayer{
			history: historyObj,
			index:   tarPathIdx,
			tree:    image.trees[layerIdx],
			tarPath: manifest.LayerTarPaths[tarPathIdx],
		}
		image.layers[layerIdx].history.Size = uint64(tree.FileSize)

		tarPathIdx++
	}

	// 计算空间利用率
	efficiency, inefficiencies := filetree.Efficiency(image.trees)

	var sizeBytes, userSizeBytes uint64
	layers := make([]Layer, len(image.layers))
	for i, v := range image.layers {
		layers[i] = v
		sizeBytes += v.Size()
		if i != 0 {
			userSizeBytes += v.Size()
		}
	}

	var wastedBytes uint64
	for idx := 0; idx < len(inefficiencies); idx++ {
		fileData := inefficiencies[len(inefficiencies)-1-idx]
		wastedBytes += uint64(fileData.CumulativeSize)
	}

	return &AnalysisResult{
		Layers:            layers,
		RefTrees:          image.trees,
		Efficiency:        efficiency,
		UserSizeByes:      userSizeBytes,
		SizeBytes:         sizeBytes,
		WastedBytes:       wastedBytes,
		WastedUserPercent: float64(float64(wastedBytes) / float64(userSizeBytes)),
		Inefficiencies:    inefficiencies,
	}, nil
}

func (image *dockerImageAnalyzer) processLayerTar(name string, layerIdx uint, reader *tar.Reader) error {
	tree := filetree.NewFileTree()
	tree.Name = name

	fileInfos, err := image.getFileList(reader)
	if err != nil {
		return err
	}

	for _, element := range fileInfos {
		tree.FileSize += uint64(element.Size)
		//fmt.Printf("TreeSize: %d\n", tree.Size)
		tree.AddPath(element.Path, element)
	}

	image.layerMap[tree.Name] = tree
	return nil
}

func (image *dockerImageAnalyzer) getFileList(tarReader *tar.Reader) ([]filetree.FileInfo, error){
	var files []filetree.FileInfo
	
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println(err)
			utils.Exit(1)
		}
		
		name := header.Name

		// Typeflag是标题项的类型
		// 根据名称中是否存在尾随斜杠，零值将自动提升为typereg或typedir。
		switch header.Typeflag {

		// Type 'g' is used by the PAX format to store key-value records that
		// are relevant to all subsequent files.
		// This package only supports parsing and composing such headers,
		// but does not currently support persisting the global state across files.
		// pax格式使用类型“g”存储与所有后续文件相关的键值记录。此包仅支持分析和撰写此类头，但当前不支持跨文件持久化全局状态。
		// TypeXGlobalHeader = 'g'
		case tar.TypeXGlobalHeader:
			return nil, fmt.Errorf("unexptected tar file: (XGlobalHeader): type=%v name=%s", header.Typeflag, name)

		// Type 'x' is used by the PAX format to store key-value records that
		// are only relevant to the next file.
		// This package transparently handles these types.
		// pax格式使用类型“x”来存储只与下一个文件相关的键值记录。
		// 这个包透明地处理这些类型。
		// TypeXHeader = 'x'
		case tar.TypeXHeader:
			return nil, fmt.Errorf("unexptected tar file (XHeader): type=%v name=%s", header.Typeflag, name)
		default:
			// name填充FileInfo.Path
			files = append(files, filetree.NewFileInfo(tarReader, header, name))
		}
	}
	return files, nil
}