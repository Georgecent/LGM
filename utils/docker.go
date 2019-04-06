package utils

import (
	"os"
	"os/exec"
	"strings"
)

// RunDockerCmd 在当前tty中运行给定的docker命令
func RunDockerCmd(cmdStr string, args ...string) error {
	allArgs := cleanArgs(append([]string{cmdStr},args...))

	cmd := exec.Command("docker", allArgs...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// cleanArgs 从给定的字符串集中删除空白字段
func cleanArgs(s []string) []string {
	var r []string
	for _, str := range s{
		if str != ""{
			r = append(r, strings.Trim(str, ""))
		}
	}
	return r
}
