package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
)

type Rule struct {
	Biz    string
	Path   string
	Regexp string
}

type Watcher interface {
	Listen(rule *Rule, taskChan chan fsnotify.FileEvent) error
	SetLogger(logger *log.LogMgr) Watcher
}

const (
	Fsnotify  = "fsnotify"
	Fspolling = "fspolling"
)
