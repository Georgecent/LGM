package ui

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"strings"
)

// StatusController 保存用于填充最底部窗格的UI对象和数据模型。 具体而言，该面板向用户显示了一组可能在窗口和当前选定窗格中执行的操作。
type StatusController struct {
	Name string
	gui  *gocui.Gui
	view *gocui.View
}

// NewStatusController 创建附加到全局[gocui]屏幕对象的新视图对象.
func NewStatusController(name string, gui *gocui.Gui) (controller *StatusController) {
	controller = new(StatusController)

	// populate main fields
	controller.Name = name
	controller.gui = gui

	return controller
}

// Setup 在全局[gocui]视图对象的上下文中初始化UI关注点。
func (controller *StatusController) Setup(v *gocui.View, header *gocui.View) error {

	// set controller options
	controller.view = v
	controller.view.Frame = false

	controller.Render()

	return nil
}

// IsVisible 指示状态视图窗格当前是否已初始化。
func (controller *StatusController) IsVisible() bool {
	if controller == nil {
		return false
	}
	return true
}

// CursorDown 在“详细信息”窗格中向下移动光标（当前指示为“无”）。
func (controller *StatusController) CursorDown() error {
	return nil
}

// CursorUp 在“详细信息”窗格中向上移动光标（当前指示为“无”）。
func (controller *StatusController) CursorUp() error {
	return nil
}

// Update 刷新状态对象以便将来进行渲染（当前不执行任何操作）。
func (controller *StatusController) Update() error {
	return nil
}

// Render 将状态对象刷新到屏幕。
func (controller *StatusController) Render() error {
	controller.gui.Update(func(g *gocui.Gui) error {
		controller.view.Clear()
		fmt.Fprintln(controller.view, controller.KeyHelp()+Controllers.lookup[controller.gui.CurrentView().Name()].KeyHelp()+Formatting.StatusNormal("▏"+strings.Repeat(" ", 1000)))

		return nil
	})
	// todo: blerg
	return nil
}

// KeyHelp 指示用户在选择当前窗格时可以执行的所有操作。
func (controller *StatusController) KeyHelp() string {
	return renderStatusOption(GlobalKeybindings.quit[0].String(), "Quit", false) +
		renderStatusOption(GlobalKeybindings.toggleView[0].String(), "Switch view", false) +
		renderStatusOption(GlobalKeybindings.filterView[0].String(), "Filter", Controllers.Filter.IsVisible())
}
