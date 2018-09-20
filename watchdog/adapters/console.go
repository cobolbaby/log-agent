package watchdog

import (
	"github.com/cobolbaby/log-agent/watchdog"
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
	// time.Sleep(time.Second) // 停顿一秒
	for _, v := range files {
		this.logger.Info("%s : %s", v.LastOp.Op, v.Filepath)
	}
	return nil
}
