package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"errors"
	"github.com/astaxie/beego/cache"
	"os"
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

func (this *FspollingWatcher) Listen(rule *fsnotify.Rule, taskChan chan *fsnotify.Event) error {
	fi, err := os.Stat(rule.MonitPath)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New("暂不支持监控单一文件")
	}

	go func() {
		bm, _ := cache.NewCache("file", `{"CachePath":"./.cache","FileSuffix":".txt","DirectoryLevel":"2", "EmbedExpiry":"0"}`)
		for {
			this.logger.Info("[FspollingWatcher] %s LoopScan Start, Path: %s", rule.Biz, rule.MonitPath)
			affectedNum := 0

			fsnotify.WalkDir(rule, 1, func(e *fsnotify.Event) error {
				if e.IsDir {
					return nil
				}
				// 检测文件是否变更
				if bm.IsExist(e.Name) && bm.Get(e.Name) == e.ModTime.String() {
					return nil
				}
				// 完善事件信息, 交给下游处理
				e.Biz = rule.Biz
				e.RootPath = rule.RootPath
				taskChan <- e

				affectedNum++
				return nil
			})

			this.logger.Info("[FspollingWatcher] %s LoopScan End, AffectedNum: %d", rule.Biz, affectedNum)
			time.Sleep(this.interval)
		}
	}()

	return nil
}
