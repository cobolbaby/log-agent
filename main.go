package main

import (
	// 需在此处添加代码。[1]
	"fmt"
	"./cmd"
	"os"
)

const Usage = `
  Usage:
    throttle <ops> [<duration>]
    throttle -h | --help
    throttle --version
  Options:
    -h, --help        output help information
    -v, --version     output version
`

func init() {

}

func main() {
	// 依据传入参数来决定是执行start/status/stop
	// os.Args 提供原始命令行参数访问功能。注意，切片中的第一个参数是该程序的路径，并且 os.Args[1:]保存所有程序的的参数。
	args := os.Args
	if len(args) < 2 {
		fmt.Println(Usage)
		os.Exit(1)
	}
	switch args[1] {
	case "help", "-h":
		fmt.Println(Usage)
	case "start":
		cmd.Start()
	case "stop":
		cmd.Stop()
	case "status":
		cmd.Status()
	case "install":
		cmd.Install()
	case "uninstall":
		cmd.Uninstall()
	case "test":
		cmd.Test()
	default:
		fmt.Println(Usage)
	}
}
