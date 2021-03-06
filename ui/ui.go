package ui

import (
	"errors"
	"LGM/filetree"
	"LGM/image"
	"LGM/keybinding"
	"LGM/utils"
	"fmt"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const debug = false


// debugPrint 将给定的字符串写入调试窗格（如果启用了调试窗格）
func debugPrint(s string) {
	if debug && Controllers.Tree != nil && Controllers.Tree.gui != nil {
		v, _ := Controllers.Tree.gui.View("debug")
		if v != nil {
			if len(v.BufferLines()) > 20 {
				v.Clear()
			}
			_, _ = fmt.Fprintln(v, s)
		}
	}
}


// Formatting 定义用于设置UI节格式的标准函数。
var Formatting struct {
	// 具有0个方法的接口称为空接口。它表示为interface{}。由于空接口有0个方法，所有类型都实现了空接口。
	Header                func(...interface{}) string
	Selected              func(...interface{}) string
	StatusSelected        func(...interface{}) string
	StatusNormal          func(...interface{}) string
	StatusControlSelected func(...interface{}) string
	StatusControlNormal   func(...interface{}) string
	CompareTop            func(...interface{}) string
	CompareBottom         func(...interface{}) string
}

var GlobalKeybindings struct {
	quit       []keybinding.Key
	// 切换视图
	toggleView []keybinding.Key
	// 过滤视图
	filterView []keybinding.Key
}

// Controllers 包含所有呈现的UI窗格
var Controllers struct {
	Tree    *FileTreeController
	Layer   *LayerController
	Status  *StatusController
	Filter  *FilterController
	Details *DetailsController
	lookup  map[string]View
}

// View 定义可渲染终端屏幕窗格。
type View interface {
	Setup(*gocui.View, *gocui.View) error
	CursorDown() error
	CursorUp() error
	Render() error
	Update() error
	KeyHelp() string
	IsVisible() bool
}

// toggleView 在file view和layer view之间切换并重新渲染屏幕。
func toggleView(g *gocui.Gui, v *gocui.View) (err error) {
	if v == nil || v.Name() == Controllers.Layer.Name {
		_, err = g.SetCurrentView(Controllers.Tree.Name)
	} else {
		_, err = g.SetCurrentView(Controllers.Layer.Name)
	}
	Update()
	Render()
	return err
}

// toggleFilterView 显示/隐藏文件树筛选器窗格。
func toggleFilterView(g *gocui.Gui, v *gocui.View) error {
	// delete all user input from the tree view
	Controllers.Filter.view.Clear()
	Controllers.Filter.view.SetCursor(0, 0)

	// toggle hiding
	Controllers.Filter.hidden = !Controllers.Filter.hidden

	if !Controllers.Filter.hidden {
		_, err := g.SetCurrentView(Controllers.Filter.Name)
		if err != nil {
			return err
		}
		Update()
		Render()
	} else {
		toggleView(g, v)
	}

	return nil
}

// CursorDown 在当前选定的gocui窗格中向下移动光标，根据需要滚动屏幕。
func CursorDown(g *gocui.Gui, v *gocui.View) error {
	return CursorStep(g, v, 1)
}

// CursorUp 在当前选定的gocui窗格中向上移动光标，根据需要滚动屏幕。
func CursorUp(g *gocui.Gui, v *gocui.View) error {
	return CursorStep(g, v, -1)
}

// quit是当用户点击 Ctrl+C 时调用的gocui回调
func quit(g *gocui.Gui, v *gocui.View) error {

	// profileObj.Stop()
	// onExit()

	return gocui.ErrQuit
}

// keyBindings 注册全局按键操作，在任何窗格中有效。
func keyBindings(g *gocui.Gui) error {
	for _, key := range GlobalKeybindings.quit {
		if err := g.SetKeybinding("", key.Value, key.Modifier, quit); err != nil {
			return err
		}
	}

	for _, key := range GlobalKeybindings.toggleView {
		if err := g.SetKeybinding("", key.Value, key.Modifier, toggleView); err != nil {
			return err
		}
	}

	for _, key := range GlobalKeybindings.filterView {
		if err := g.SetKeybinding("", key.Value, key.Modifier, toggleFilterView); err != nil {
			return err
		}
	}

	return nil
}

// isNewView 确定是否已根据给定的一组错误（有点矫揉造作）创建视图
func isNewView(errs ...error) bool {
	for _, err := range errs {
		if err == nil {
			return false
		}
		if err != nil && err != gocui.ErrUnknownView {
			return false
		}
	}
	return true
}

// layout 定义窗口窗格大小的定义以及彼此之间的位置关系。这在应用程序启动时以及屏幕尺寸更改时调用。
func layout(g *gocui.Gui) error {
	// TODO: this logic should be refactored into an abstraction that takes care of the math for us

	maxX, maxY := g.Size()
	fileTreeSplitRatio := viper.GetFloat64("filetree.pane-width")
	if fileTreeSplitRatio >= 1 || fileTreeSplitRatio <= 0 {
		logrus.Errorf("invalid config value: 'filetree.pane-width' should be 0 < value < 1, given '%v'", fileTreeSplitRatio)
		fileTreeSplitRatio = 0.5
	}
	splitCols := int(float64(maxX) * (1.0 - fileTreeSplitRatio))
	debugWidth := 0
	if debug {
		debugWidth = maxX / 4
	}
	debugCols := maxX - debugWidth
	bottomRows := 1
	headerRows := 2

	filterBarHeight := 1
	statusBarHeight := 1

	statusBarIndex := 1
	filterBarIndex := 2

	layersHeight := len(Controllers.Layer.Layers) + headerRows + 1 // layers + header + base image layer row
	maxLayerHeight := int(0.75 * float64(maxY))
	if layersHeight > maxLayerHeight {
		layersHeight = maxLayerHeight
	}

	var view, header *gocui.View
	var viewErr, headerErr, err error

	if Controllers.Filter.hidden {
		bottomRows--
		filterBarHeight = 0
	}

	// Debug pane
	if debug {
		if _, err := g.SetView("debug", debugCols, -1, maxX, maxY-bottomRows); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
		}
	}

	// Layers
	view, viewErr = g.SetView(Controllers.Layer.Name, -1, -1+headerRows, splitCols, layersHeight)
	header, headerErr = g.SetView(Controllers.Layer.Name+"header", -1, -1, splitCols, headerRows)
	if isNewView(viewErr, headerErr) {
		Controllers.Layer.Setup(view, header)

		if _, err = g.SetCurrentView(Controllers.Layer.Name); err != nil {
			return err
		}
		// since we are selecting the view, we should rerender to indicate it is selected
		Controllers.Layer.Render()
	}

	// Details
	view, viewErr = g.SetView(Controllers.Details.Name, -1, -1+layersHeight+headerRows, splitCols, maxY-bottomRows)
	header, headerErr = g.SetView(Controllers.Details.Name+"header", -1, -1+layersHeight, splitCols, layersHeight+headerRows)
	if isNewView(viewErr, headerErr) {
		Controllers.Details.Setup(view, header)
	}

	// Filetree
	offset := 0
	if !Controllers.Tree.vm.ShowAttributes {
		offset = 1
	}
	view, viewErr = g.SetView(Controllers.Tree.Name, splitCols, -1+headerRows-offset, debugCols, maxY-bottomRows)
	header, headerErr = g.SetView(Controllers.Tree.Name+"header", splitCols, -1, debugCols, headerRows-offset)
	if isNewView(viewErr, headerErr) {
		Controllers.Tree.Setup(view, header)
	}
	Controllers.Tree.onLayoutChange()

	// Status Bar
	view, viewErr = g.SetView(Controllers.Status.Name, -1, maxY-statusBarHeight-statusBarIndex, maxX, maxY-(statusBarIndex-1))
	if isNewView(viewErr, headerErr) {
		Controllers.Status.Setup(view, nil)
	}

	// Filter Bar
	view, viewErr = g.SetView(Controllers.Filter.Name, len(Controllers.Filter.headerStr)-1, maxY-filterBarHeight-filterBarIndex, maxX, maxY-(filterBarIndex-1))
	header, headerErr = g.SetView(Controllers.Filter.Name+"header", -1, maxY-filterBarHeight-filterBarIndex, len(Controllers.Filter.headerStr), maxY-(filterBarIndex-1))
	if isNewView(viewErr, headerErr) {
		Controllers.Filter.Setup(view, header)
	}

	return nil
}

// Update 刷新状态对象以便将来进行渲染。
func Update() {
	for _, view := range Controllers.lookup {
		view.Update()
	}
}

// Render 将状态对象刷新到屏幕。
func Render() {
	for _, view := range Controllers.lookup {
		if view.IsVisible() {
			view.Render()
		}
	}
}

// renderStatusOption 将键帮助绑定格式化为标题对。
func renderStatusOption(control, title string, selected bool) string {
	if selected {
		return Formatting.StatusSelected("▏") + Formatting.StatusControlSelected(control) + Formatting.StatusSelected(" "+title+" ")
	} else {
		return Formatting.StatusNormal("▏") + Formatting.StatusControlNormal(control) + Formatting.StatusNormal(" "+title+" ")
	}
}

// Run is the UI entrypoint.
func Run(analysis *image.AnalysisResult, cache filetree.TreeCache) {
	Formatting.Selected = color.New(color.ReverseVideo, color.Bold).SprintFunc()
	Formatting.Header = color.New(color.Bold).SprintFunc()
	Formatting.StatusSelected = color.New(color.BgMagenta, color.FgWhite).SprintFunc()
	Formatting.StatusNormal = color.New(color.ReverseVideo).SprintFunc()
	Formatting.StatusControlSelected = color.New(color.BgMagenta, color.FgWhite, color.Bold).SprintFunc()
	Formatting.StatusControlNormal = color.New(color.ReverseVideo, color.Bold).SprintFunc()
	Formatting.CompareTop = color.New(color.BgMagenta).SprintFunc()
	Formatting.CompareBottom = color.New(color.BgGreen).SprintFunc()

	var err error
	GlobalKeybindings.quit, err = keybinding.ParseAll(viper.GetString("keybinding.quit"))
	if err != nil {
		logrus.Error(err)
	}
	GlobalKeybindings.toggleView, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-view"))
	if err != nil {
		logrus.Error(err)
	}
	GlobalKeybindings.filterView, err = keybinding.ParseAll(viper.GetString("keybinding.filter-files"))
	if err != nil {
		logrus.Error(err)
	}

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		logrus.Error(err)
	}
	utils.SetUi(g)
	defer g.Close()

	Controllers.lookup = make(map[string]View)

	Controllers.Layer = NewLayerController("side", g, analysis.Layers)
	Controllers.lookup[Controllers.Layer.Name] = Controllers.Layer

	Controllers.Tree = NewFileTreeController("main", g, filetree.StackTreeRange(analysis.RefTrees, 0, 0), analysis.RefTrees, cache)
	Controllers.lookup[Controllers.Tree.Name] = Controllers.Tree

	Controllers.Status = NewStatusController("status", g)
	Controllers.lookup[Controllers.Status.Name] = Controllers.Status

	Controllers.Filter = NewFilterController("command", g)
	Controllers.lookup[Controllers.Filter.Name] = Controllers.Filter

	Controllers.Details = NewDetailsController("details", g, analysis.Efficiency, analysis.Inefficiencies)
	Controllers.lookup[Controllers.Details.Name] = Controllers.Details

	g.Cursor = false

	g.SetManagerFunc(layout)

	// perform the first update and render now that all resources have been loaded
	Update()
	Render()

	if err := keyBindings(g); err != nil {
		logrus.Error(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		logrus.Error(err)
	}
	utils.Exit(0)
}

// 将光标移动给定的步距，将原点设置为新光标线
func CursorStep(g *gocui.Gui, v *gocui.View, step int) error {
	cx, cy := v.Cursor()

	// if there isn't a next line
	line, err := v.Line(cy + step)
	if err != nil {
		// todo: handle error
	}
	if len(line) == 0 {
		return errors.New("unable to move the cursor, empty line")
	}
	// SetCursor设置视图在给定点相对于视图的光标位置。它检查位置是否有效。
	if err := v.SetCursor(cx, cy+step); err != nil {
		// Origin返回视图的原点位置。
		ox, oy := v.Origin()
		// setorigin设置视图内部缓冲区的起始位置，因此缓冲区从该点开始打印，这意味着它与起始点视图链接。它可以用来实现水平和垂直滚动，只需增加或减少Ox和Oy。
		if err := v.SetOrigin(ox, oy+step); err != nil {
			return err
		}
	}
	return nil
}