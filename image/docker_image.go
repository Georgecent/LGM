package image

import (
	"LGM/filetree"
	"LGM/utils"
	"archive/tar"
	"context"
	"fmt"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
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
			}else if strings.HasSuffix(name, "json") {	//HasSuffix测试字符串是否以.json后缀结尾。
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

}

func (image *dockerImageAnalyzer) processLayerTar(name string, layerIdx uint, reader *tar.Reader) error {
	tree := filetree.NewFileTree()
	tree.Name = name

	fileInfos, err := image.getFileList(reader)
	if err != nil {
		return err
	}

	for _, element := range fileInfos{
		tree.FileSize += uint64(element.Size)
	}
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

		switch header.Typeflag {
		case tar.TypeXGlobalHeader:
			return nil, fmt.Errorf("unexptected tar file: (XGlobalHeader): type=%v name=%s", header.Typeflag, name)
		case tar.TypeXHeader:
			return nil, fmt.Errorf("unexptected tar file (XHeader): type=%v name=%s", header.Typeflag, name)
		default:
			files = append(files, filetree.NewFileInfo(tarReader, header, name))
		}
	}
	return files, nil
}