package utils

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/sirupsen/logrus"
	"os"
)

var ui *gocui.Gui

func SetUi(g *gocui.Gui) {
	ui = g
}

func PrintAndExit(args ...interface{})  {
	logrus.Println(args...)
	CleanUp()
	fmt.Println(args...)
	os.Exit(1)
}

func Exit(rc int) {
	CleanUp()
	os.Exit(rc)
}

func CleanUp() {
	if ui != nil {
		ui.Close()
	}
}