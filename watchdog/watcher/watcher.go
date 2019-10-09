package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
)

type Watcher interface {
	Listen(rule *fsnotify.Rule, taskChan chan *fsnotify.Event)
	SetLogger(logger log.Logger) Watcher
}

const (
	FS_NOTIFY = "fsnotify"
	FS_POLL   = "fspolling"
)
