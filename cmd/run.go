package cmd

import (
	"dc-agent-go/plugins"
	. "dc-agent-go/utils"
	"dc-agent-go/watchdog"
	"dc-agent-go/watchdog/lib/log"
	"github.com/kardianos/osext"
	"path/filepath"
)

func Run() {
	cfg := ConfigMgr()

	fullexecpath, _ := osext.Executable()
	execdir, _ := filepath.Split(fullexecpath)
	logger := log.NewLogMgr(filepath.Join(execdir, "logs"))

	agentSwitch := cfg.Section("").Key("switch").MustBool()
	if !agentSwitch {
		logger.Fatal("LogAgent Monitor Switch Close :)")
	}
	if cfg.Section("").Key("hostname").Value() == "localhost" {
		logger.Fatal("Please Modify hostname in logagent.ini :)")
	}

	watchdog.NewWatchdog().
		SetHost(cfg.Section("").Key("hostname").Value()).
		SetLogger(logger).
		LoadPlugins(plugins.Autoload()).
		Run()
}
