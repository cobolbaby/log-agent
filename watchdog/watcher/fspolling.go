package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"io/ioutil"
	"path/filepath"
	"sync"
)

var (
	sema   = make(chan struct{}, 255) // sema is a counting semaphore for limiting concurrency
	syncWg sync.WaitGroup
)

type FspollingWatcher struct{}

func NewFspollingWatcher() *FspollingWatcher {
	return &FspollingWatcher{}
}

func (this *FspollingWatcher) Listen(rule *Rule) error {

	for _, dir := range rule.Rules {
		syncWg.Add(1)
		go func(dir string) {
			defer syncWg.Done()

			walkDir(dir, func(path string) {
				rule.DelayQueueChan <- fsnotify.FileEvent{
					Biz:  rule.Biz,
					Op:   "LOAD",
					Name: path,
				}
			})
		}(dir)
	}
	syncWg.Wait()
	return nil
}

func walkDir(dir string, cb func(path string)) error {
	sema <- struct{}{} // acquire token
	defer func() {     // release token
		<-sema
	}()
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			syncWg.Add(1)
			go func(path string) {
				defer syncWg.Done()

				walkDir(path, cb)
			}(filepath.Join(dir, e.Name()))
		} else {
			cb(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}
