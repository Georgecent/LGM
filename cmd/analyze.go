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
	"fmt"
	"github.com/spf13/cobra"
)

// doAnalyzeCmd 获取docker镜像tag、摘要或id，并将图像分析显示在屏幕上
func doAnalyzeCmd(cmd *cobra.Command, args []string)  {
	defer utils.CleanUp()
	if len(args) == 0 {
		// PersistentFlags返回在当前命令中专门设置的持久FlagSet。
		printVersionFlag, err := cmd.PersistentFlags().GetBool("version")
		if err == nil && printVersionFlag {
			PrintVersion(cmd, args)
			return
		}

		fmt.Println("No image argument given")
		cmd.Help()
		utils.Exit(1)
	}

	userImage := args[0]
	if userImage == "" {
		fmt.Println("No image argument given")
		cmd.Help()
		utils.Exit(1)
	}

	initLogging()

	runtime.Run(runtime.Options{
		ImageId:      userImage,
		ExportFile:   exportFile,
		CiConfigFile: ciConfigFile,
	})
}
