package ui

import (
	"LGM/image"
	"LGM/keybinding"
	"LGM/utils"
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/lunixbochs/vtclean"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

type CompareType int

const (
	CompareLayer CompareType = iota
	CompareAll
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

// NewLayerController 创建一个附加全局[gocui]屏幕对象的新视图对象。
func NewLayerController(name string, gui *gocui.Gui, layers []image.Layer) (controller *LayerController) {
	controller = new(LayerController)

	// 填充主要字段
	controller.Name = name
	controller.gui = gui
	controller.Layers = layers

	// 显示汇总的更改
	switch mode := viper.GetBool("layer.show-aggregated-changes"); mode {
	case true:
		controller.CompareMode = CompareAll
	case false:
		controller.CompareMode = CompareLayer
	default:
		utils.PrintAndExit(fmt.Sprintf("unknown layer.show-aggregated-changes value: %v", mode))
	}

	var err error
	controller.keybindingCompareAll, err = keybinding.ParseAll(viper.GetString("keybinding.compare-all"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingCompareLayer, err = keybinding.ParseAll(viper.GetString("keybinding.compare-layer"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingPageUp, err = keybinding.ParseAll(viper.GetString("keybinding.page-up"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingPageDown, err = keybinding.ParseAll(viper.GetString("keybinding.page-down"))
	if err != nil {
		logrus.Error(err)
	}

	return controller
}

// Setup 在全局[gocui]视图对象的上下文中初始化UI关注点。
func (controller *LayerController) Setup(v *gocui.View, header *gocui.View) error {

	// set controller options
	controller.view = v
	// 如果Editable为true，则键击将添加到光标位置的视图内部缓冲区中。
	controller.view.Editable = false
	// 如果Wrap为true，则写入此View的内容会在长度超过其宽度时自动换行。 如果为true，则忽略视图的x-origin。
	controller.view.Wrap = false
	// 如果Frame为true，则将在视图周围绘制边框。
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
	if err := controller.gui.SetKeybinding(controller.Name, gocui.KeyArrowRight, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return controller.CursorDown() }); err != nil {
		return err
	}
	if err := controller.gui.SetKeybinding(controller.Name, gocui.KeyArrowLeft, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return controller.CursorUp() }); err != nil {
		return err
	}

	for _, key := range controller.keybindingPageUp {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.PageUp() }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingPageDown {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.PageDown() }); err != nil {
			return err
		}
	}

	for _, key := range controller.keybindingCompareLayer {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.setCompareMode(CompareLayer) }); err != nil {
			return err
		}
	}

	for _, key := range controller.keybindingCompareAll {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.setCompareMode(CompareAll) }); err != nil {
			return err
		}
	}

	return controller.Render()
}

// height 获取当前窗格的高度（考虑到由于标题而丢失的空间）。
func (controller *LayerController) height() uint {
	_, height := controller.view.Size()
	return uint(height - 1)
}

// IsVisible 指示层视图窗格当前是否已初始化。
func (controller *LayerController) IsVisible() bool {
	if controller == nil {
		return false
	}
	return true
}

// PageDown 移到下一页，将光标放在顶部
func (controller *LayerController) PageDown() error {
	step := int(controller.height()) + 1
	targetLayerIndex := controller.LayerIndex + step

	if targetLayerIndex > len(controller.Layers) {
		step -= targetLayerIndex - (len(controller.Layers) - 1)
		targetLayerIndex = controller.LayerIndex + step
	}

	if step > 0 {
		err := CursorStep(controller.gui, controller.view, step)
		if err == nil {
			controller.SetCursor(controller.LayerIndex + step)
		}
	}
	return nil
}

// PageUp 移到上一页，将光标放在上面
func (controller *LayerController) PageUp() error {
	step := int(controller.height()) + 1
	targetLayerIndex := controller.LayerIndex - step

	if targetLayerIndex < 0 {
		step += targetLayerIndex
		targetLayerIndex = controller.LayerIndex - step
	}

	if step > 0 {
		err := CursorStep(controller.gui, controller.view, -step)
		if err == nil {
			controller.SetCursor(controller.LayerIndex - step)
		}
	}
	return nil
}

// CursorDown 在图层窗格中向下移动光标（选择更高的图层）。
func (controller *LayerController) CursorDown() error {
	if controller.LayerIndex < len(controller.Layers) {
		err := CursorDown(controller.gui, controller.view)
		if err == nil {
			controller.SetCursor(controller.LayerIndex + 1)
		}
	}
	return nil
}

// CursorUp 在图层窗格中向上移动光标（选择较低的图层）。
func (controller *LayerController) CursorUp() error {
	if controller.LayerIndex > 0 {
		err := CursorUp(controller.gui, controller.view)
		if err == nil {
			controller.SetCursor(controller.LayerIndex - 1)
		}
	}
	return nil
}

// SetCursor 重置光标并根据给定的层索引确定文件树视图的方向。
func (controller *LayerController) SetCursor(layer int) error {
	controller.LayerIndex = layer
	Controllers.Tree.setTreeByLayer(controller.getCompareIndexes())
	Controllers.Details.Render()
	controller.Render()

	return nil
}

// currentLayer 返回当前选定的图层对象。
func (controller *LayerController) currentLayer() image.Layer {
	return controller.Layers[(len(controller.Layers)-1)-controller.LayerIndex]
}

// setCompareMode 将单层比较与聚合比较之间的层比较进行切换。
func (controller *LayerController) setCompareMode(compareMode CompareType) error {
	controller.CompareMode = compareMode
	Update()
	Render()
	return Controllers.Tree.setTreeByLayer(controller.getCompareIndexes())
}

// getCompareIndexes 确定用于比较的层边界（基于当前比较模式）
func (controller *LayerController) getCompareIndexes() (bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop int) {
	bottomTreeStart = controller.CompareStartIndex
	topTreeStop = controller.LayerIndex

	if controller.LayerIndex == controller.CompareStartIndex {
		bottomTreeStop = controller.LayerIndex
		topTreeStart = controller.LayerIndex
	} else if controller.CompareMode == CompareLayer {
		bottomTreeStop = controller.LayerIndex - 1
		topTreeStart = controller.LayerIndex
	} else {
		bottomTreeStop = controller.CompareStartIndex
		topTreeStart = controller.CompareStartIndex + 1
	}

	return bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop
}

// renderCompareBar 返回给定层的格式化字符串。
func (controller *LayerController) renderCompareBar(layerIdx int) string {
	bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop := controller.getCompareIndexes()
	result := "  "

	if layerIdx >= bottomTreeStart && layerIdx <= bottomTreeStop {
		result = Formatting.CompareBottom("  ")
	}
	if layerIdx >= topTreeStart && layerIdx <= topTreeStop {
		result = Formatting.CompareTop("  ")
	}

	return result
}

// Update 刷新状态对象以便将来进行渲染（当前不执行任何操作）。
func (controller *LayerController) Update() error {
	controller.ImageSize = 0
	for idx := 0; idx < len(controller.Layers); idx++ {
		controller.ImageSize += controller.Layers[idx].Size()
	}
	return nil
}

// Render 将状态对象刷新到屏幕。图层窗格报告：
// 1. the layers of the image + metadata
// 2. the current selected image
func (controller *LayerController) Render() error {

	// indicate when selected
	title := "Layers"
	if controller.gui.CurrentView() == controller.view {
		title = "● " + title
	}

	controller.gui.Update(func(g *gocui.Gui) error {
		// update header
		controller.header.Clear()
		width, _ := g.Size()
		headerStr := fmt.Sprintf("[%s]%s\n", title, strings.Repeat("─", width*2))
		// headerStr += fmt.Sprintf("Cmp "+image.LayerFormat, "Layer Digest", "Size", "Command")
		headerStr += fmt.Sprintf("Cmp"+image.LayerFormat, "Size", "Command")
		fmt.Fprintln(controller.header, Formatting.Header(vtclean.Clean(headerStr, false)))

		// update contents
		controller.view.Clear()
		for revIdx := len(controller.Layers) - 1; revIdx >= 0; revIdx-- {
			layer := controller.Layers[revIdx]
			idx := (len(controller.Layers) - 1) - revIdx

			layerStr := layer.String()
			compareBar := controller.renderCompareBar(idx)

			if idx == controller.LayerIndex {
				fmt.Fprintln(controller.view, compareBar+" "+Formatting.Selected(layerStr))
			} else {
				fmt.Fprintln(controller.view, compareBar+" "+layerStr)
			}

		}
		return nil
	})
	return nil
}

// KeyHelp 指示用户在选择当前窗格时可以执行的所有操作。
func (controller *LayerController) KeyHelp() string {
	return renderStatusOption(controller.keybindingCompareLayer[0].String(), "Show layer changes", controller.CompareMode == CompareLayer) +
		renderStatusOption(controller.keybindingCompareAll[0].String(), "Show aggregated changes", controller.CompareMode == CompareAll)
}
