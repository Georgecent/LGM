package ui

import (
	"LGM/filetree"
	"LGM/keybinding"
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/lunixbochs/vtclean"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"regexp"
	"strings"
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

// NewFileTreeController 创建一个附加全局[gocui]屏幕对象的新视图对象。
func NewFileTreeController(name string, gui *gocui.Gui, tree *filetree.FileTree, refTrees []*filetree.FileTree, cache filetree.TreeCache) (controller *FileTreeController) {
	controller = new(FileTreeController)

	// populate main fields
	controller.Name = name
	controller.gui = gui
	controller.vm = NewFileTreeViewModel(tree, refTrees, cache)

	var err error
	controller.keybindingToggleCollapse, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-collapse-dir"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingToggleCollapseAll, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-collapse-all-dir"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingToggleAttributes, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-filetree-attributes"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingToggleAdded, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-added-files"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingToggleRemoved, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-removed-files"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingToggleModified, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-modified-files"))
	if err != nil {
		logrus.Error(err)
	}

	controller.keybindingToggleUnchanged, err = keybinding.ParseAll(viper.GetString("keybinding.toggle-unchanged-files"))
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
func (controller *FileTreeController) Setup(v *gocui.View, header *gocui.View) error {

	// set controller options
	controller.view = v
	controller.view.Editable = false
	controller.view.Wrap = false
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
	if err := controller.gui.SetKeybinding(controller.Name, gocui.KeyArrowLeft, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return controller.CursorLeft() }); err != nil {
		return err
	}
	if err := controller.gui.SetKeybinding(controller.Name, gocui.KeyArrowRight, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return controller.CursorRight() }); err != nil {
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
	for _, key := range controller.keybindingToggleCollapse {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleCollapse() }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingToggleCollapseAll {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleCollapseAll() }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingToggleAttributes {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleAttributes() }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingToggleAdded {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleShowDiffType(filetree.Added) }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingToggleRemoved {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleShowDiffType(filetree.Removed) }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingToggleModified {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleShowDiffType(filetree.Changed) }); err != nil {
			return err
		}
	}
	for _, key := range controller.keybindingToggleUnchanged {
		if err := controller.gui.SetKeybinding(controller.Name, key.Value, key.Modifier, func(*gocui.Gui, *gocui.View) error { return controller.toggleShowDiffType(filetree.Unchanged) }); err != nil {
			return err
		}
	}

	_, height := controller.view.Size()
	controller.vm.Setup(0, height)
	controller.Update()
	controller.Render()

	return nil
}

// IsVisible 指示文件树视图窗格当前是否已初始化
func (controller *FileTreeController) IsVisible() bool {
	if controller == nil {
		return false
	}
	return true
}

// resetCursor 将光标移回缓冲区的顶部并转换为缓冲区的顶部。
func (controller *FileTreeController) resetCursor() {
	controller.view.SetCursor(0, 0)
	controller.vm.resetCursor()
}

// setTreeByLayer 通过堆叠指示的图像层文件树来填充视图模型。
func (controller *FileTreeController) setTreeByLayer(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop int) error {
	err := controller.vm.setTreeByLayer(bottomTreeStart, bottomTreeStop, topTreeStart, topTreeStop)
	if err != nil {
		return err
	}
	// controller.resetCursor()

	controller.Update()
	return controller.Render()
}

// CursorDown 向下移动光标并呈现视图。
// 注意：我们不能使用gocui缓冲区，因为任何状态更改都需要将整个树写入缓冲区。
// 相反，我们保持树形字符串的上限和下限以进行渲染，并且仅将此范围刷新到视图缓冲区中。 当树大小很大时，这会快得多。
func (controller *FileTreeController) CursorDown() error {
	if controller.vm.CursorDown() {
		return controller.Render()
	}
	return nil
}

// CursorUp 向上移动光标并呈现视图。
// 注意：我们不能使用gocui缓冲区，因为任何状态更改都需要将整个树写入缓冲区。
// 相反，我们保持树形字符串的上限和下限以进行渲染，并且仅将此范围刷新到视图缓冲区中。 当树大小很大时，这会快得多。
func (controller *FileTreeController) CursorUp() error {
	if controller.vm.CursorUp() {
		return controller.Render()
	}
	return nil
}

// CursorLeft 将光标向上移动，直到我们到达父节点或树的顶部
func (controller *FileTreeController) CursorLeft() error {
	err := controller.vm.CursorLeft(filterRegex())
	if err != nil {
		return err
	}
	controller.Update()
	return controller.Render()
}

// CursorRight 如果需要，可以进入扩展目录的目录
func (controller *FileTreeController) CursorRight() error {
	err := controller.vm.CursorRight(filterRegex())
	if err != nil {
		return err
	}
	controller.Update()
	return controller.Render()
}

// PageDown 移动到下一页，将光标置于顶部
func (controller *FileTreeController) PageDown() error {
	err := controller.vm.PageDown()
	if err != nil {
		return err
	}
	return controller.Render()
}

// PageUp 移动到上一页，将光标置于顶部
func (controller *FileTreeController) PageUp() error {
	err := controller.vm.PageUp()
	if err != nil {
		return err
	}
	return controller.Render()
}

// getAbsPositionNode 确定所选屏幕光标在文件树中的位置，返回所选的FileNode。
func (controller *FileTreeController) getAbsPositionNode() (node *filetree.FileNode) {
	return controller.vm.getAbsPositionNode(filterRegex())
}

// toggleCollapse 将折叠/展开选定的FileNode。
func (controller *FileTreeController) toggleCollapse() error {
	err := controller.vm.toggleCollapse(filterRegex())
	if err != nil {
		return err
	}
	controller.Update()
	return controller.Render()
}

// toggleCollapseAll 将折叠/展开所有目录。
func (controller *FileTreeController) toggleCollapseAll() error {
	err := controller.vm.toggleCollapseAll()
	if err != nil {
		return err
	}
	controller.Update()
	return controller.Render()
}

// toggleAttributes 将显示/隐藏文件属性
func (controller *FileTreeController) toggleAttributes() error {
	err := controller.vm.toggleAttributes()
	if err != nil {
		return err
	}
	// we need to render the changes to the status pane as well
	Update()
	Render()
	return nil
}

// toggleShowDiffType 将在filetree窗格中显示/隐藏选定的DiffType。
func (controller *FileTreeController) toggleShowDiffType(diffType filetree.DiffType) error {
	controller.vm.toggleShowDiffType(diffType)
	// we need to render the changes to the status pane as well
	Update()
	Render()
	return nil
}

// onLayoutChange UI框架调用onLayoutChange以通知视图模型新的屏幕尺寸
func (controller *FileTreeController) onLayoutChange() error {
	controller.Update()
	return controller.Render()
}

// filterRegex 将返回正则表达式对象以匹配用户的过滤器输入。
func filterRegex() *regexp.Regexp {
	if Controllers.Filter == nil || Controllers.Filter.view == nil {
		return nil
	}
	filterString := strings.TrimSpace(Controllers.Filter.view.Buffer())
	if len(filterString) == 0 {
		return nil
	}

	regex, err := regexp.Compile(filterString)
	if err != nil {
		return nil
	}

	return regex
}

// Update 刷新状态对象以便将来进行渲染。
func (controller *FileTreeController) Update() error {
	var width, height int

	if controller.view != nil {
		width, height = controller.view.Size()
	} else {
		// 在设置TUI之前，可能没有可供参考的控制器。使用整个屏幕作为参考。
		width, height = controller.gui.Size()
	}
	// height should account for the header
	return controller.vm.Update(filterRegex(), width, height-1)
}

// Render 将状态对象（文件树）刷新到窗格。
func (controller *FileTreeController) Render() error {
	title := "Current Layer Contents"
	if Controllers.Layer.CompareMode == CompareAll {
		title = "Aggregated Layer Contents"
	}

	// indicate when selected
	if controller.gui.CurrentView() == controller.view {
		title = "● " + title
	}

	controller.gui.Update(func(g *gocui.Gui) error {
		// update the header
		controller.header.Clear()
		width, _ := g.Size()
		headerStr := fmt.Sprintf("[%s]%s\n", title, strings.Repeat("─", width*2))
		if controller.vm.ShowAttributes {
			headerStr += fmt.Sprintf(filetree.AttributeFormat+" %s", "P", "ermission", "UID:GID", "Size", "Filetree")
		}

		fmt.Fprintln(controller.header, Formatting.Header(vtclean.Clean(headerStr, false)))

		// update the contents
		controller.view.Clear()
		controller.vm.Render()
		fmt.Fprint(controller.view, controller.vm.mainBuf.String())

		return nil
	})
	return nil
}

// KeyHelp 指示用户在选择当前窗格时可以执行的所有操作。
func (controller *FileTreeController) KeyHelp() string {
	return renderStatusOption(controller.keybindingToggleCollapse[0].String(), "Collapse dir", false) +
		renderStatusOption(controller.keybindingToggleCollapseAll[0].String(), "Collapse all dir", false) +
		renderStatusOption(controller.keybindingToggleAdded[0].String(), "Added", !controller.vm.HiddenDiffTypes[filetree.Added]) +
		renderStatusOption(controller.keybindingToggleRemoved[0].String(), "Removed", !controller.vm.HiddenDiffTypes[filetree.Removed]) +
		renderStatusOption(controller.keybindingToggleModified[0].String(), "Modified", !controller.vm.HiddenDiffTypes[filetree.Changed]) +
		renderStatusOption(controller.keybindingToggleUnchanged[0].String(), "Unmodified", !controller.vm.HiddenDiffTypes[filetree.Unchanged]) +
		renderStatusOption(controller.keybindingToggleAttributes[0].String(), "Attributes", controller.vm.ShowAttributes)
}
