package ui

import (
	"LGM/keybinding"
	"github.com/jroimartin/gocui"
)

// FileTreeController 保存用于填充右窗格的UI对象和数据模型。 特别是显示所选图层或聚合文件ASCII树的窗格。
type FileTreeController struct {
	Name   string
	gui    *gocui.Gui
	view   *gocui.View
	header *gocui.View
	vm     *FileTreeViewModel

	keybindingToggleCollapse    []keybinding.Key
	keybindingToggleCollapseAll []keybinding.Key
	keybindingToggleAttributes  []keybinding.Key
	keybindingToggleAdded       []keybinding.Key
	keybindingToggleRemoved     []keybinding.Key
	keybindingToggleModified    []keybinding.Key
	keybindingToggleUnchanged   []keybinding.Key
	keybindingPageDown          []keybinding.Key
	keybindingPageUp            []keybinding.Key
}
