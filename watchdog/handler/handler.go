package handler

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"time"
)

type FileMeta struct {
	Filepath     string
	Pack         string
	Dirname      string
	Filename     string
	Size         int64 // 字节
	Ext          string
	CreateTime   time.Time
	ModifyTime   time.Time
	Content      []byte
	LastOp       fsnotify.FileEvent
	BackUpTime   time.Time // 文件备份时间
	Checksum     string
	Compress     bool
	CompressSize int64
	Reference    string // 保留字段
	Host         string // 文件溯源
}

type WatchdogHandler interface {
	Handle(file FileMeta) error
	Rollback(file FileMeta) error
	SetLogger(logger log.Logger)
}
