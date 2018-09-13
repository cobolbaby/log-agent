package main

import (
	// 需在此处添加代码。[1]
	Cmd "./command"
	Util "./util"
	"fmt"
	"os"
)

const INFO_HELP = `
	help message1
	help message2
`

var appCfg map[string]string

func init() {
	// 加载配置文件(加载本地配置文件)
	filename := "./conf/agent.ini"
	err := Util.LoadCfg("INI", filename)
	if err != nil {
		fmt.Printf("Load the configuration error. [error=%v]\n", err)
		panic("Failed to Load configuration.Please make sure that the configuration exists")
	}
	fmt.Printf("loadConf()...\n")
}

func main() {
	// 依据传入参数来决定是执行start/status/stop
	// os.Args 提供原始命令行参数访问功能。注意，切片中的第一个参数是该程序的路径，并且 os.Args[1:]保存所有程序的的参数。
	args := os.Args
	if len(args) < 2 {
		fmt.Println(INFO_HELP)
		return
	}
	switch args[1] {
	case "help", "-h":
		fmt.Println(INFO_HELP)
	case "start":
		Cmd.Start(appCfg)
	case "stop":
		Cmd.Stop(appCfg)
	case "status":
		Cmd.Status(appCfg)
	default:
		fmt.Println(INFO_HELP)
	}
}
