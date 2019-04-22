package ui

import (
	"LGM/filetree"
	"LGM/image"
	"LGM/keybinding"
	"LGM/utils"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Formatting defines standard functions for formatting UI sections.
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


}