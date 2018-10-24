package main

import (
	// 需在此处添加代码。[1]
	"dc-agent-go/cmd"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/osext"
	"github.com/kardianos/service"
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

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	cmd.Run()
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func getConfigPath() (string, error) {
	fullexecpath, err := osext.Executable()
	if err != nil {
		return "", err
	}

	dir, _ := filepath.Split(fullexecpath)
	return filepath.Join(dir, "conf/logagent.ini"), nil
}

func main() {

	//服务的配置信息
	configPath, err := getConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	cfg := &service.Config{
		Name:      "LogFileAgent",
		Arguments: []string{"-c", configPath},
	}
	prg := &program{}
	// 构建服务对象
	s, err := service.New(prg, cfg)
	if err != nil {
		log.Fatal(err)
	}
	// logger 用于记录系统日志
	logger, err := s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	// 依据传入参数来决定是执行start/status/stop
	// os.Args 提供原始命令行参数访问功能。注意，切片中的第一个参数是该程序的路径，并且 os.Args[1:]保存所有程序的的参数。
	args := os.Args
	if len(args) < 2 {
		log.Println(Usage)
		os.Exit(1)
	}
	switch args[1] {
	case "-c":
		if len(args) < 3 {
			log.Println(Usage)
			os.Exit(1)
		}
		os.Setenv("LOGAGENT_CONF_PATH", args[2])
		if err = s.Run(); err != nil {
			logger.Error(err)
		}
	case "-t":
		if len(args) < 3 {
			log.Println(Usage)
			os.Exit(1)
		}
		os.Setenv("LOGAGENT_CONF_PATH", args[2])
		cmd.Test()
	case "start", "stop", "restart", "install", "uninstall":
		// Ps: 需要拥有管理员的权限
		if err = service.Control(s, os.Args[1]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Println(Usage)
	}
}
