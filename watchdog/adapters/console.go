package watchdog

import (
	"github.com/cobolbaby/log-agent/watchdog"
	"time"
)

type ConsoleAdapter struct {
	Name   string
	Config *ConsoleAdapterCfg
	logger watchdog.Logger
}

type ConsoleAdapterCfg struct {
}

func (this *ConsoleAdapter) SetLogger(logger watchdog.Logger) watchdog.WatchdogAdapter {
	this.logger = logger
	return this
}

func (this *ConsoleAdapter) Handle(files []watchdog.FileMeta) error {
	// getFileMeta
	// write the filename to stdout
	this.logger.Info("[ConsoleAdapter] -------------  %s  -------------", time.Now().Format("2006/1/2 15:04:05"))
	for _, v := range files {
		this.logger.Info("%s FILE: %s", v.LastOp.Op, v.Filepath)
	}
	return nil
}
