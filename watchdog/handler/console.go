package handler

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
)

type ConsoleAdapter struct {
	Name     string
	Config   *ConsoleAdapterCfg
	logger   log.Logger
	Priority uint8
}

type ConsoleAdapterCfg struct {
}

func NewConsoleAdapter() (WatchdogHandler, error) {
	return &ConsoleAdapter{
		Name: "Console",
	}, nil
}

func (this *ConsoleAdapter) SetLogger(logger log.Logger) {
	this.logger = logger
}

func (this *ConsoleAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *ConsoleAdapter) Handle(fi FileMeta) error {
	// write the filename to stdout
	this.logger.Debugf("[ConsoleAdapter] %s %s", fi.Filepath, fi.LastOp.Op)
	return nil
}

func (this *ConsoleAdapter) Rollback(fi FileMeta) error {
	return nil
}
