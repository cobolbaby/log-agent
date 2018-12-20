package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
)

type FsnotifyWatcher struct {
	logger *log.LogMgr
}

func NewFsnotifyWatcher() Watcher {
	return &FsnotifyWatcher{}
}

func (this *FsnotifyWatcher) SetLogger(logger *log.LogMgr) Watcher {
	this.logger = logger
	return this
}

func (this *FsnotifyWatcher) Listen(rule *Rule, taskChan chan fsnotify.FileEvent) error {
	// 当前仅支持监控单一目录
	monitorDir := rule.Rules[0]

	watcher, err := fsnotify.NewRecursiveWatcher()
	if err != nil {
		return err
	}
	// defer watcher.Close()

	go watcher.NotifyFsEvent(monitorDir, func(err error, e fsnotify.FileEvent) {
		if err != nil {
			this.logger.Error("[NotifyFsEvent] %s", err)
			return
		}
		// 过滤
		if e.Op == "Create" || e.Op == "Write" {
			e.Biz = rule.Biz
			taskChan <- e
		}
	})

	err = watcher.RecursiveAdd(monitorDir)
	if err != nil {
		return err
	}

	return nil
}
