package cmd

import (
	"github.com/cobolbaby/log-agent/plugins"
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/kardianos/osext"
	"log"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
)

func Run() {
	cfg := ConfigMgr()

	fullexecpath, _ := osext.Executable()
	execdir, _ := filepath.Split(fullexecpath)

	agentSwitch := cfg.Section("").Key("switch").MustBool()
	if !agentSwitch {
		log.Fatal("LogAgent Monitor Switch Close :)")
	}
	hostname := cfg.Section("").Key("hostname").Value()
	if hostname == "localhost" {
		log.Fatal("Please Modify hostname in logagent.ini :)")
	}
	dataPath := cfg.Section("").Key("data").Value()
	if dataPath == "" {
		dataPath = filepath.Join(execdir, "data")
	}
	logPath := cfg.Section("").Key("logs").Value()
	if logPath == "" {
		logPath = filepath.Join(execdir, "logs")
	}

	watchdog.NewWatchdog().
		SetHost(hostname).
		SetLogPath(logPath).
		SetDataPath(dataPath).
		LoadPlugins(plugins.Autoload()).
		Run()

	// 启动程序监控
	go http.ListenAndServe(":12345", nil)

	// TODO:推送心跳信息
}
