package main

import (
	// 需在此处添加代码。[1]
	"fmt"
	"github.com/cobolbaby/log-agent/cmd"
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
	case "--help", "-h":
		fmt.Println(Usage)
	case "-f":
		os.Setenv("LOGAGENT_CONF_PATH", args[2])
		cmd.Start()
	case "stop", "-q":
		cmd.Stop()
	case "status", "-s":
		cmd.Status()
	case "install":
		cmd.Install()
	case "uninstall":
		cmd.Uninstall()
	case "test", "-t":
		cmd.Test()
	default:
		fmt.Println(Usage)
	}
}
