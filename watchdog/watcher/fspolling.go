package watcher

import (
	"github.com/astaxie/beego/cache"
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

	bm, err := cache.NewCache("file", `{"CachePath":"./.cache","FileSuffix":".txt","DirectoryLevel":2, "EmbedExpiry":0}`)
	if err != nil {
		return err
	}

	for _, dir := range rule.Rules {
		syncWg.Add(1)
		go func(dir string) {
			defer syncWg.Done()

			walkDir(dir, func(e fsnotify.FileEvent) {
				// 检测文件的变更情况
				if bm.IsExist(e.Name) && bm.Get(e.Name) == e.ModTime.String() {
					return
				}
				e.Biz = rule.Biz
				rule.DelayQueueChan <- e
			})
		}(dir)
	}
	syncWg.Wait()
	return nil
}

func walkDir(dir string, cb func(e fsnotify.FileEvent)) error {
	sema <- struct{}{} // acquire token
	defer func() {     // release token
		<-sema
	}()
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			cb(fsnotify.FileEvent{
				Op:      "LOAD",
				Name:    filepath.Join(dir, e.Name()),
				ModTime: e.ModTime(),
			})
			continue
		}
		syncWg.Add(1)
		go func(path string) {
			defer syncWg.Done()

			walkDir(path, cb)
		}(filepath.Join(dir, e.Name()))
	}
	return nil
}
