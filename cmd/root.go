// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"LGM/utils"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
)

var cfgFile string
var exportFile string
var ciConfigFile string


// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "LGM [IMAGE]",
	Short: "Docker Image Visualizer & Explorer",
	Long: `LGM is a command line tool that can be run on Ubuntu/Debian, RHEL/Centos, Arch Linux and other platforms. 
It is mainly used to mine Docker images, analyze layer content, and help reduce the size of Docker images.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Args: cobra.MaximumNArgs(1),
	Run:  doAnalyzeCmd,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.LGM.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.Flags().StringVarP(&exportFile, "json", "j", "", "Skip the interactive TUI and write the layer analysis statistics to a given file.")
	rootCmd.Flags().StringVar(&ciConfigFile, "ci-config", ".LGM-ci", "If CI=true in the environment, use the given yaml to drive validation rules.")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	filepathToCfg := getCfgFile(cfgFile)
	viper.SetConfigFile(filepathToCfg)

	viper.SetDefault("log.level", log.InfoLevel.String())
	viper.SetDefault("log.path", "./LGM.log")
	viper.SetDefault("log.enabled", true)
	// keybindings: status view / global
	viper.SetDefault("keybinding.quit", "ctrl+c")
	viper.SetDefault("keybinding.toggle-view", "tab")
	viper.SetDefault("keybinding.filter-files", "ctrl+f, ctrl+slash")
	// keybindings: layer view
	viper.SetDefault("keybinding.compare-all", "ctrl+a")
	viper.SetDefault("keybinding.compare-layer", "ctrl+l")
	// keybindings: filetree view
	viper.SetDefault("keybinding.toggle-collapse-dir", "space")
	viper.SetDefault("keybinding.toggle-collapse-all-dir", "ctrl+space")
	viper.SetDefault("keybinding.toggle-filetree-attributes", "ctrl+b")
	viper.SetDefault("keybinding.toggle-added-files", "ctrl+a")
	viper.SetDefault("keybinding.toggle-removed-files", "ctrl+r")
	viper.SetDefault("keybinding.toggle-modified-files", "ctrl+m")
	viper.SetDefault("keybinding.toggle-unchanged-files", "ctrl+u")
	viper.SetDefault("keybinding.page-up", "pgup")
	viper.SetDefault("keybinding.page-down", "pgdn")

	viper.SetDefault("diff.hide", "")

	viper.SetDefault("layer.show-aggregated-changes", false)

	viper.SetDefault("filetree.collapse-dir", false)
	viper.SetDefault("filetree.pane-width", 0.5)
	viper.SetDefault("filetree.show-attributes", true)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}


}

// initLogging 使用格式化程序和位置设置日志对象
func initLogging()  {
	var logFileObj *os.File
	var err error

	if viper.GetBool("log.enabled"){
		logFileObj, err = os.OpenFile(viper.GetString("log.path"), os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0644)
	}else {
		// Discard 是一个io.Writer 接口，调用它的Write 方法将不做任何事情  并且始终成功返回。
		log.SetOutput(ioutil.Discard)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	Formatter := new(log.TextFormatter)
	Formatter.DisableTimestamp = true
	log.SetFormatter(Formatter)

	level, err := log.ParseLevel(viper.GetString("log.level"))
	if err != nil{
		fmt.Fprintln(os.Stderr, err)
	}

	log.SetLevel(level)
	log.SetOutput(logFileObj)
	log.Debug("Starting LGM...")
}

// getCfgFile checks for config file in paths from xdg specs
// and in $HOME/.config/LGM/ directory
// defaults to $HOME/.LGM.yaml
func getCfgFile(fromFlag string) string {
	if fromFlag != "" {
		return fromFlag
	}

	home ,err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		utils.Exit(0)
	}

	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	xdgDirs := os.Getenv("XDG_CONFIG_DIRS")
	xdgPaths := append([]string{xdgHome}, strings.Split(xdgDirs,":")...)
	// path.Join增加子路径
	allDir := append(xdgPaths, path.Join(home, ".config"))

	for _, val := range allDir {
		file := findInPath(val)
		if len(file) > 0 {
			return file
		}
	}
	return path.Join(home, ".LGM.yaml")
}

// findInPath returns first "*.yaml" file in path's subdirectory "LGM"
// if not found returns empty string
func findInPath(pathTo string) string {
	directory := path.Join(pathTo, "LGM")
	files ,err := ioutil.ReadDir(directory)

	if err != nil {
		return ""
	}

	for _, file := range files{
		fileName := file.Name()
		if path.Ext(fileName) == ".yaml" {
			return path.Join(directory, fileName)
		}
	}
	return ""
}
