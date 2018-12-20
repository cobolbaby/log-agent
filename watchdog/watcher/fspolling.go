package watcher

import (
	"dc-agent-go/watchdog/lib/fsnotify"
	"dc-agent-go/watchdog/lib/log"
	"github.com/astaxie/beego/cache"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type FspollingWatcher struct {
	logger   *log.LogMgr
	interval time.Duration
}

func NewFspollingWatcher(interval time.Duration) Watcher {
	return &FspollingWatcher{
		interval: interval,
	}
}

func (this *FspollingWatcher) SetLogger(logger *log.LogMgr) Watcher {
	this.logger = logger
	return this
}

func (this *FspollingWatcher) Listen(rule *Rule, taskChan chan fsnotify.FileEvent) error {
	// 当前仅支持监控单一目录
	monitorDir := rule.Rules[0]
	if _, err := ioutil.ReadDir(monitorDir); err != nil {
		return err
	}

	bm, _ := cache.NewCache("file", `{"CachePath":"./.cache","FileSuffix":".txt","DirectoryLevel":2, "EmbedExpiry":0}`)

	go func() {
		for {
			this.logger.Info("[FspollingWatcher] %s LoopScan Start, Path: %s", rule.Biz, monitorDir)
			affectedNum := 0
			walkDir(monitorDir, func(e fsnotify.FileEvent) {
				// 检测文件的变更情况
				if bm.IsExist(e.Name) && bm.Get(e.Name) == e.ModTime.String() {
					return
				}
				e.Biz = rule.Biz
				e.MonitorDir = monitorDir
				taskChan <- e
				affectedNum++
			})
			this.logger.Info("[FspollingWatcher] %s LoopScan End, AffectedNum: %d", rule.Biz, affectedNum)
			// 可以考虑加部分抖动
			time.Sleep(this.interval)
		}
	}()

	return nil
}

func walkDir(monitorDir string, cb func(e fsnotify.FileEvent)) {
	dir, _ := os.Open(monitorDir)
	defer dir.Close()
	// unsorted file list
	fis, _ := dir.Readdir(-1)
	for _, fi := range fis {
		if !fi.IsDir() {
			cb(fsnotify.FileEvent{
				ModTime: fi.ModTime(),
				Op:      "LOAD",
				Name:    filepath.Join(monitorDir, fi.Name()),
			})
			continue
		}
		walkDir(filepath.Join(monitorDir, fi.Name()), cb)
	}
}
