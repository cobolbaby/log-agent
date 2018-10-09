package handler

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"time"
)

type RabbitmqAdapter struct {
	Name     string
	Config   *RabbitmqAdapterCfg
	logger   log.Logger
	Priority uint8
}

type RabbitmqAdapterCfg struct {
}

func (this *RabbitmqAdapter) SetLogger(logger log.Logger) {
	this.logger = logger
}

func (this *FileAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *RabbitmqAdapter) Handle(fi FileMeta) error {
	// write the filename to stdout
	this.logger.Info("[RabbitmqAdapter] -------------  %s  -------------", time.Now().Format("2006/1/2 15:04:05"))
	this.logger.Info("%s FILE: %s", fi.LastOp.Op, fi.Filepath)
	return nil
}

func (this *RabbitmqAdapter) Rollback(fi FileMeta) error {
	return nil
}
