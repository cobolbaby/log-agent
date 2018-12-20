package main

import (
	// 需在此处添加代码。[1]
	"github.com/cobolbaby/log-agent/cmd"
	"github.com/kardianos/osext"
	"github.com/kardianos/service"
	"log"
	"os"
	"path/filepath"
	// "runtime"
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

var (
	GIT_COMMIT string
	BUILD_TIME string
	GO_VERSION string
)

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

func getDefaultConfigPath() (string, error) {
	fullexecpath, err := osext.Executable()
	if err != nil {
		return "", err
	}

	dir, _ := filepath.Split(fullexecpath)
	return filepath.Join(dir, "conf", "logagent.ini"), nil
}

func main() {
	// runtime.GOMAXPROCS(runtime.NumCPU() / 2.0)

	//服务的配置信息
	configPath, err := getDefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	cfg := &service.Config{
		Name:      "DCAgent",
		Arguments: []string{"-c", configPath},
	}
	// Interface 接口
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
		log.Fatal(Usage)
	}
	switch args[1] {
	case "-v", "version":
		log.Println("Git Commit: " + GIT_COMMIT)
		log.Println("Build Time: " + BUILD_TIME)
		log.Println("Go Version: " + GO_VERSION)
	case "-c":
		if len(args) < 3 {
			log.Fatal(Usage)
		}
		if err = s.Run(); err != nil {
			logger.Error(err)
		}
	case "-t":
		if len(args) < 3 {
			log.Fatal(Usage)
		}
		cmd.Test()
	case "start", "stop", "restart", "install", "uninstall":
		// Ps: 需要拥有管理员的权限
		if err = service.Control(s, os.Args[1]); err != nil {
			log.Fatal(err)
		}
	case "status":
		// TODO:查看当前程序运行状态
	default:
		log.Fatal(Usage)
	}
}
