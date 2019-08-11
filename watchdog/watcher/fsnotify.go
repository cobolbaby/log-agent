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

func (this *FsnotifyWatcher) Listen(rule *fsnotify.Rule, taskChan chan *fsnotify.Event) error {

	watcher, err := fsnotify.NewRecursiveWatcher()
	if err != nil {
		return err
	}
	// defer watcher.Close()

	go watcher.NotifyFsEvent(rule, func(e *fsnotify.Event, err error) {
		if err != nil {
			this.logger.Error("[fsnotify.NotifyFsEvent] %s", err)
			return
		}
		this.logger.Info("[fsnotify.NotifyFsEvent] %s %s", e.Op, e.Name)
		if e.Op == "CREATE" || e.Op == "WRITE" {
			e.Biz = rule.Biz
			e.RootPath = rule.RootPath
			taskChan <- e
		}
	})

	return watcher.RecursiveAdd(rule)
}
