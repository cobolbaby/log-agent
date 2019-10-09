package watcher

import (
	"bytes"
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger"
	"time"
)

const (
	FS_POLL_INTERVAL = 10 * time.Minute // 文件系统轮询时间间隔
)

type FspollingWatcher struct {
	logger log.Logger
	db     *badger.DB
}

func NewFspollingWatcher(db *badger.DB) Watcher {
	return &FspollingWatcher{
		db: db,
	}
}

func (this *FspollingWatcher) SetLogger(logger log.Logger) Watcher {
	this.logger = logger
	return this
}

func (this *FspollingWatcher) Listen(rule *fsnotify.Rule, taskChan chan *fsnotify.Event) {

	go func() {
		// 目录遍历不受递归层级的限制，作用是在保证高效实时监听的情况下，避免影响历史数据导入
		r := new(fsnotify.Rule)
		*r = *rule
		r.MaxNestingLevel = 0
		for {
			this.logger.Infof("Start to scan %s, Path: %s", rule.Biz, rule.MonitPath)

			affectedNum := 0
			err := fsnotify.WalkDir(r, 1, func(e *fsnotify.Event) error {
				if e.IsDir {
					return nil
				}
				// 检测文件是否变更
				if t, _ := e.ModTime.GobEncode(); this.isSaved([]byte(e.Name), t) {
					return nil
				}
				// 完善事件信息, 交给下游处理
				e.Biz = rule.Biz
				e.RootPath = rule.RootPath
				taskChan <- e

				affectedNum++
				return nil
			})
			if err != nil {
				this.logger.Errorf("The error occured during polling filesystem: %s", err)
			}

			this.logger.Infof("End to scan %s, AffectedNum: #%d", rule.Biz, affectedNum)
			time.Sleep(FS_POLL_INTERVAL)
		}
	}()

}

func (this *FspollingWatcher) isSaved(k []byte, v []byte) bool {
	// check if the same key=value already exists
	err := this.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(k)
		if err != nil { // not found
			return err
		}
		return item.Value(func(val []byte) error {
			if bytes.Equal(val, v) { // already saved
				return nil
			}
			errmsg := fmt.Sprintf("%s is already updated", v)
			return errors.New(errmsg)
		})
	})
	return err == nil
}
