package watchdog

import (
	"github.com/cobolbaby/log-agent/watchdog"
	"time"
)

type RabbitmqAdapter struct {
	Name   string
	Config *RabbitmqAdapterCfg
	logger watchdog.Logger
}

type RabbitmqAdapterCfg struct {
}

func (this *RabbitmqAdapter) SetLogger(logger watchdog.Logger) watchdog.WatchdogAdapter {
	this.logger = logger
	return this
}

func (this *RabbitmqAdapter) Handle(fi watchdog.FileMeta) error {
	// write the filename to stdout
	this.logger.Info("[RabbitmqAdapter] -------------  %s  -------------", time.Now().Format("2006/1/2 15:04:05"))
	this.logger.Info("%s FILE: %s", fi.LastOp.Op, fi.Filepath)
	return nil
}

func (this *RabbitmqAdapter) Rollback(fi watchdog.FileMeta) error {
	return nil
}
