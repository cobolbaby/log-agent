package watcher

import (
	"dc-agent-go/watchdog/lib/fsnotify"
	"dc-agent-go/watchdog/lib/log"
)

type Rule struct {
	Biz   string
	Rules []string
}

type Watcher interface {
	Listen(rule *Rule, taskChan chan fsnotify.FileEvent) error
	SetLogger(logger *log.LogMgr) Watcher
}

const (
	Fsnotify  = "fsnotify"
	Fspolling = "fspolling"
)
