package main

import (
	// 需在此处添加代码。[1]
	"fmt"
	"github.com/cobolbaby/log-agent/command"
	"os"
)

const INFO_HELP = `
	help message1
	help message2
`

func init() {

}

func main() {
	// 依据传入参数来决定是执行start/status/stop
	// os.Args 提供原始命令行参数访问功能。注意，切片中的第一个参数是该程序的路径，并且 os.Args[1:]保存所有程序的的参数。
	args := os.Args
	if len(args) < 2 {
		fmt.Println(INFO_HELP)
		os.Exit(1)
	}
	switch args[1] {
	case "help", "-h":
		fmt.Println(INFO_HELP)
	case "start":
		command.Start()
	case "stop":
		command.Stop()
	case "status":
		command.Status()
	case "install":
		command.Install()
	case "uninstall":
		command.Uninstall()
	default:
		fmt.Println(INFO_HELP)
	}
}
