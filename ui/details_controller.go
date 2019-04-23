package ui

import (
	"LGM/filetree"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jroimartin/gocui"
	"github.com/lunixbochs/vtclean"
	"strconv"
	"strings"
)

// DetailsController 保存用于填充左下窗格的UI对象和数据模型。 特别是显示图层详细信息和图像统计信息的窗格。
type DetailsController struct {
	Name           string
	gui            *gocui.Gui
	view           *gocui.View
	header         *gocui.View
	efficiency     float64
	inefficiencies filetree.EfficiencySlice
}

// NewDetailsController 创建附加到全局[gocui]屏幕对象的新视图对象。
func NewDetailsController(name string, gui *gocui.Gui, efficiency float64, inefficiencies filetree.EfficiencySlice) (controller *DetailsController) {
	controller = new(DetailsController)

	// populate main fields
	controller.Name = name
	controller.gui = gui
	controller.efficiency = efficiency
	controller.inefficiencies = inefficiencies

	return controller
}

// Setup 在全局[gocui]视图对象的上下文中初始化UI关注点。
func (controller *DetailsController) Setup(v *gocui.View, header *gocui.View) error {

	// set controller options
	controller.view = v
	controller.view.Editable = false
	controller.view.Wrap = true
	controller.view.Highlight = false
	controller.view.Frame = false

	controller.header = header
	controller.header.Editable = false
	controller.header.Wrap = false
	controller.header.Frame = false

	// set keybindings
	if err := controller.gui.SetKeybinding(controller.Name, gocui.KeyArrowDown, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return controller.CursorDown() }); err != nil {
		return err
	}
	if err := controller.gui.SetKeybinding(controller.Name, gocui.KeyArrowUp, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return controller.CursorUp() }); err != nil {
		return err
	}

	return controller.Render()
}

// IsVisible 指示详细信息视图窗格当前是否已初始化。
func (controller *DetailsController) IsVisible() bool {
	if controller == nil {
		return false
	}
	return true
}

// CursorDown 在“详细信息”窗格中向下移动光标（当前指示为“无”）。
func (controller *DetailsController) CursorDown() error {
	return CursorDown(controller.gui, controller.view)
}

// CursorUp 在“详细信息”窗格中向上移动光标（当前指示为“无”）。
func (controller *DetailsController) CursorUp() error {
	return CursorUp(controller.gui, controller.view)
}

// Update 刷新状态对象以便将来进行渲染。
func (controller *DetailsController) Update() error {
	return nil
}

// Render 将状态对象刷新到屏幕。
// 详细信息窗格报告：
//	1.当前所选图层的命令字符串
//	2.图像效率得分
//	3.估计浪费的图像空间
//	4.低效文件分配列表
func (controller *DetailsController) Render() error {
	currentLayer := Controllers.Layer.currentLayer()

	var wastedSpace int64

	template := "%5s  %12s  %-s\n"
	inefficiencyReport := fmt.Sprintf(Formatting.Header(template), "Count", "Total Space", "Path")

	height := 100
	if controller.view != nil {
		_, height = controller.view.Size()
	}

	for idx := 0; idx < len(controller.inefficiencies); idx++ {
		data := controller.inefficiencies[len(controller.inefficiencies)-1-idx]
		wastedSpace += data.CumulativeSize

		// todo: make this report scrollable
		if idx < height {
			inefficiencyReport += fmt.Sprintf(template, strconv.Itoa(len(data.Nodes)), humanize.Bytes(uint64(data.CumulativeSize)), data.Path)
		}
	}

	imageSizeStr := fmt.Sprintf("%s %s", Formatting.Header("Total Image size:"), humanize.Bytes(Controllers.Layer.ImageSize))
	effStr := fmt.Sprintf("%s %d %%", Formatting.Header("Image efficiency score:"), int(100.0*controller.efficiency))
	wastedSpaceStr := fmt.Sprintf("%s %s", Formatting.Header("Potential wasted space:"), humanize.Bytes(uint64(wastedSpace)))

	controller.gui.Update(func(g *gocui.Gui) error {
		// update header
		controller.header.Clear()
		width, _ := controller.view.Size()

		layerHeaderStr := fmt.Sprintf("[Layer Details]%s", strings.Repeat("─", width-15))
		imageHeaderStr := fmt.Sprintf("[Image Details]%s", strings.Repeat("─", width-15))

		fmt.Fprintln(controller.header, Formatting.Header(vtclean.Clean(layerHeaderStr, false)))

		// update contents
		controller.view.Clear()
		fmt.Fprintln(controller.view, Formatting.Header("Digest: ")+currentLayer.Id())
		// TODO: add back in with controller model
		// fmt.Fprintln(view.view, Formatting.Header("Tar ID: ")+currentLayer.TarId())
		fmt.Fprintln(controller.view, Formatting.Header("Command:"))
		fmt.Fprintln(controller.view, currentLayer.Command())

		fmt.Fprintln(controller.view, "\n"+Formatting.Header(vtclean.Clean(imageHeaderStr, false)))

		fmt.Fprintln(controller.view, imageSizeStr)
		fmt.Fprintln(controller.view, wastedSpaceStr)
		fmt.Fprintln(controller.view, effStr+"\n")

		fmt.Fprintln(controller.view, inefficiencyReport)
		return nil
	})
	return nil
}

// KeyHelp 表示用户在选择当前窗格时可以执行的所有操作（当前不执行任何操作）。
func (controller *DetailsController) KeyHelp() string {
	return "TBD"
}