package handler

import (
	"bytes"
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
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
	LastOp       *fsnotify.Event
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
	SetLogger(logger log.Logger)
	GetPriority() uint8
}

const (
	Cassandra = "cassandra"
	Console   = "console"
	File      = "file"
	RABBITMQ  = "rabbitmq"
	KAFKA     = "kafka"
)

// GBK转化为UTF8
func GBKToUTF8(src string) (string, error) {
	I := bytes.NewReader([]byte(src))
	O := transform.NewReader(I, simplifiedchinese.GBK.NewDecoder())
	res, e := ioutil.ReadAll(O)
	if e != nil {
		return "", e
	}
	return string(res), nil
}
