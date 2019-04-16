package filetree

type TreeCacheKey struct {
	bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop int
}

type TreeCache struct {
	refTree []*FileTree
	cache map[TreeCacheKey]*FileTree
}

func NewFileTreeCache(refTrees []*FileTree) TreeCache {
	return TreeCache{
		refTree:refTrees,
		cache:make(map[TreeCacheKey]*FileTree),
	}
}

func (cache *TreeCache) Build()  {
	var bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop int
	// case 1: layer compare (top tree SIZE is fixed (BUT floats forward), Bottom tree SIZE changes)
	for selectIdx := 0; selectIdx < len(cache.refTrees); selectIdx++ {
		bottomTreeStart = 0
		topTreeStop = selectIdx

		if selectIdx == 0 {
			bottomTreeStop = selectIdx
			topTreeStart = selectIdx
		} else {
			bottomTreeStop = selectIdx - 1
			topTreeStart = selectIdx
		}

		cache.Get(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop)
	}

	// case 2: aggregated compare (bottom tree is ENTIRELY fixed, top tree SIZE changes)
	for selectIdx := 0; selectIdx < len(cache.refTrees); selectIdx++ {
		bottomTreeStart = 0
		topTreeStop = selectIdx
		if selectIdx == 0 {
			bottomTreeStop = selectIdx
			topTreeStart = selectIdx
		} else {
			bottomTreeStop = 0
			topTreeStart = 1
		}

		cache.Get(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop)
	}
}
