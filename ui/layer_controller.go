package ui

import (
	"LGM/image"
	"LGM/keybinding"
	"github.com/jroimartin/gocui"
)

// LayerController 包含用于填充左下窗格的UI对象和数据模型。 特别是显示图像图层和图层选择器的窗格。
type LayerController struct {
	Name              string
	gui               *gocui.Gui
	view              *gocui.View
	header            *gocui.View
	LayerIndex        int
	Layers            []image.Layer
	CompareMode       CompareType
	CompareStartIndex int
	ImageSize         uint64

	keybindingCompareAll   []keybinding.Key
	keybindingCompareLayer []keybinding.Key
	keybindingPageDown     []keybinding.Key
	keybindingPageUp       []keybinding.Key
}
