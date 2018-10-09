package handler

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"time"
)

type ConsoleAdapter struct {
	Name     string
	Config   *ConsoleAdapterCfg
	logger   log.Logger
	Priority int
}

type ConsoleAdapterCfg struct {
}

func (this *ConsoleAdapter) SetLogger(logger log.Logger) {
	this.logger = logger
}

func (this *ConsoleAdapter) Handle(fi FileMeta) error {
	// write the filename to stdout
	this.logger.Info("[ConsoleAdapter] -------------  %s  -------------", time.Now().Format("2006/1/2 15:04:05"))
	this.logger.Info("[ConsoleAdapter] %s %s", fi.LastOp.Op, fi.Filepath)
	return nil
}

func (this *ConsoleAdapter) Rollback(fi FileMeta) error {
	return nil
}
