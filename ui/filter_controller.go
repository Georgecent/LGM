package ui

import (
	"fmt"
	"github.com/jroimartin/gocui"
)

// FilterController 包含用于填充底行的UI对象和数据模型。 特别是允许用户按路径过滤文件树的窗格。
type FilterController struct {
	Name      string
	gui       *gocui.Gui
	view      *gocui.View
	header    *gocui.View
	headerStr string
	maxLength int
	hidden    bool
}

// NewFilterController 创建一个附加全局[gocui]屏幕对象的新视图对象。
func NewFilterController(name string, gui *gocui.Gui) (controller *FilterController) {
	controller = new(FilterController)

	// populate main fields
	controller.Name = name
	controller.gui = gui
	controller.headerStr = "Path Filter: "
	controller.hidden = true

	return controller
}

// Setup 在全局[gocui]视图对象的上下文中初始化UI关注点。
func (controller *FilterController) Setup(v *gocui.View, header *gocui.View) error {

	// set controller options
	controller.view = v
	controller.maxLength = 200
	controller.view.Frame = false
	controller.view.BgColor = gocui.AttrReverse
	controller.view.Editable = true
	controller.view.Editor = controller

	controller.header = header
	controller.header.BgColor = gocui.AttrReverse
	controller.header.Editable = false
	controller.header.Wrap = false
	controller.header.Frame = false

	controller.Render()

	return nil
}

// IsVisible 指示过滤器视图窗格当前是否已初始化
func (controller *FilterController) IsVisible() bool {
	if controller == nil {
		return false
	}
	return !controller.hidden
}

// CursorDown 在过滤器窗格中向下移动光标（当前不指示任何内容）。
func (controller *FilterController) CursorDown() error {
	return nil
}

// CursorUp 将光标向上移动到筛选器窗格中（当前不指示任何内容）。
func (controller *FilterController) CursorUp() error {
	return nil
}

// Edit 拦截文件管理器视图中的按键事件，以实时更新文件视图。
func (controller *FilterController) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if !controller.IsVisible() {
		return
	}

	cx, _ := v.Cursor()
	ox, _ := v.Origin()
	limit := ox+cx+1 > controller.maxLength
	switch {
	case ch != 0 && mod == 0 && !limit:
		v.EditWrite(ch)
	case key == gocui.KeySpace && !limit:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	}
	if Controllers.Tree != nil {
		Controllers.Tree.Update()
		Controllers.Tree.Render()
	}
}

// Update 刷新状态对象以供将来渲染（当前不执行任何操作）。
func (controller *FilterController) Update() error {
	return nil
}

// Render 将状态对象刷新到屏幕。当前这是用户路径筛选器输入。
func (controller *FilterController) Render() error {
	controller.gui.Update(func(g *gocui.Gui) error {
		// render the header
		fmt.Fprintln(controller.header, Formatting.Header(controller.headerStr))

		return nil
	})
	return nil
}

// KeyHelp 指示选择当前窗格时用户可以采取的所有可能操作。
func (controller *FilterController) KeyHelp() string {
	return Formatting.StatusControlNormal("▏Type to filter the file tree ")
}
