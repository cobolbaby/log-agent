package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"time"
)

type Rule struct {
	Biz            string
	Rules          []string
	DelayQueueChan chan fsnotify.FileEvent
	Delay          time.Duration
	TaskQueueChan  chan []fsnotify.FileEvent
}

type Watcher interface {
	Listen(rule *Rule) error
}
