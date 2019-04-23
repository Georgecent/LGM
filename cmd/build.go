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
	"LGM/runtime"
	"LGM/utils"
	"github.com/spf13/cobra"
)

// buildCmd 表示生成命令
var buildCmd = &cobra.Command{
	Use:   "build [any valid 'docker build' arguments]",
	Short: "Builds and analyzes a docker image from a Dockerfile (this is a thin wrapper for the 'docker build' command).",
	Run: doBuildCmd,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func doBuildCmd(cmd *cobra.Command, args []string) {
	defer utils.CleanUp()

	initLogging()

	runtime.Run(runtime.Options{
		BuildArgs:  args,
		ExportFile: exportFile,
	})
}