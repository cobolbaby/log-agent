package handler

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
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

func (this *RabbitmqAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *RabbitmqAdapter) Handle(fi FileMeta) error {
	// TODO:推送消息的结构体，如何标准化
	this.logger.Debugf("[RabbitmqAdapter] %s %s", fi.Filepath, fi.LastOp.Op)

	return nil
}

func (this *RabbitmqAdapter) Rollback(fi FileMeta) error {
	return nil
}
