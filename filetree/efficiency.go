package filetree

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sort"
)

// Len is required for sorting.
func (efs EfficiencySlice) Len() int {
	return len(efs)
}

// Swap operation is required for sorting.
func (efs EfficiencySlice) Swap(i, j int) {
	efs[i], efs[j] = efs[j], efs[i]
}

// Less comparison is required for sorting.
func (efs EfficiencySlice) Less(i, j int) bool {
	return efs[i].CumulativeSize < efs[j].CumulativeSize
}

// Efficiency 返回给定文件树集（层）的分数和文件集。这大致基于：
// 1. 跨层重复的文件会折扣您的分数，按文件大小加权
// 2. 删除的文件会折扣您的分数，并按原始文件大小加权
func Efficiency(trees []*FileTree) (float64, EfficiencySlice) {
	efficiencyMap := make(map[string]*EfficiencyData)
	inefficientMatches := make(EfficiencySlice, 0)
	currentTree := 0

	visitor := func(node *FileNode) error {
		path := node.Path()
		if _, ok := efficiencyMap[path]; !ok {
			efficiencyMap[path] = &EfficiencyData{
				Path:              path,
				Nodes:             make([]*FileNode, 0),
				minDiscoveredSize: -1,
			}
		}
		data := efficiencyMap[path]

		// 此节点可能已经删除了的子节点，但是，我们不会明确列出每个子节点，只列出具有累积大小的最顶层父节点。 这些操作需要在完整（堆叠）树上完成。
		// 注意：whiteout文件也可能代表目录，所以我们需要找出它以前是文件还是目录。
		var sizeBytes int64

		if node.IsWhiteout() {
			sizer := func(curNode *FileNode) error {
				sizeBytes += curNode.Data.FileInfo.Size
				return nil
			}
			stackedTree := StackTreeRange(trees, 0, currentTree-1)
			previousTreeNode, err := stackedTree.GetNode(node.Path())
			if err != nil {
				logrus.Debug(fmt.Sprintf("CurrentTree: %d : %s", currentTree, err))
			} else if previousTreeNode.Data.FileInfo.IsDir {
				err = previousTreeNode.VisitDepthChildFirst(sizer, nil)
				if err != nil {
					logrus.Errorf("unable to propagate whiteout dir: %+v", err)
				}
			}

		} else {
			sizeBytes = node.Data.FileInfo.Size
		}

		data.CumulativeSize += sizeBytes
		if data.minDiscoveredSize < 0 || sizeBytes < data.minDiscoveredSize {
			data.minDiscoveredSize = sizeBytes
		}
		data.Nodes = append(data.Nodes, node)

		if len(data.Nodes) == 2 {
			inefficientMatches = append(inefficientMatches, data)
		}

		return nil
	}
	visitEvaluator := func(node *FileNode) bool {
		return node.IsLeaf()
	}
	for idx, tree := range trees {
		currentTree = idx
		err := tree.VisitDepthChildFirst(visitor, visitEvaluator)
		if err != nil {
			logrus.Errorf("unable to propagate ref tree: %+v", err)
		}
	}

	// 计算分数
	var minimumPathSizes int64
	var discoveredPathSizes int64

	for _, value := range efficiencyMap {
		minimumPathSizes += value.minDiscoveredSize
		discoveredPathSizes += value.CumulativeSize
	}
	var score float64
	if discoveredPathSizes == 0 {
		score = 1.0
	} else {
		score = float64(minimumPathSizes) / float64(discoveredPathSizes)
	}

	sort.Sort(inefficientMatches)

	return score, inefficientMatches
}