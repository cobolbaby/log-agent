package handler

import (
	"dc-agent-go/watchdog/lib/fsnotify"
	"dc-agent-go/watchdog/lib/log"
	"time"
)

type FileMeta struct {
	Filepath     string
	Pack         string
	SubDir       string
	Filename     string
	Size         int64 // 字节
	Ext          string
	CreateTime   time.Time // 文件创建时间
	ModifyTime   time.Time // 文件修改时间
	Content      []byte
	LastOp       fsnotify.FileEvent
	Checksum     string
	Compress     bool
	CompressSize int64
	Reference    string    // 保留字段
	Host         string    // 文件溯源
	FolderTime   time.Time // 文件所在目录的创建时间
}

type WatchdogHandler interface {
	Handle(file FileMeta) error
	Rollback(file FileMeta) error
	SetLogger(logger *log.LogMgr)
	GetPriority() uint8
}

const (
	Cassandra = "cassandra"
	Console   = "console"
	File      = "file"
	RabbitMQ  = "rabbitmq"
)
