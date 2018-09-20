package watchdog

import (
	"fmt"
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
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}
