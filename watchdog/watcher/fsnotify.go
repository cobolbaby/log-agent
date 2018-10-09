package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
)

type FsnotifyWatcher struct{}

func NewFsnotifyWatcher() *FsnotifyWatcher {
	return &FsnotifyWatcher{}
}

func (this *FsnotifyWatcher) Listen(rule *Rule) error {
	watcher, err := fsnotify.NewRecursiveWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go watcher.NotifyFsEvent(func(e fsnotify.FileEvent) {
		e.Biz = rule.Biz
		rule.DelayQueueChan <- e
	})

	for _, r := range rule.Rules {
		err := watcher.RecursiveAdd(r)
		if err != nil {
			return err
		}
	}

	done := make(chan bool)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
	return nil
}