package watcher

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"os"
	"strings"
	"time"
)

type FsnotifyWatcher struct {
	logger log.Logger
	reset  bool // 是否应该进行重置
}

func NewFsnotifyWatcher() Watcher {
	return &FsnotifyWatcher{
		reset: false,
	}
}

func (this *FsnotifyWatcher) SetLogger(logger log.Logger) Watcher {
	this.logger = logger
	return this
}

func (this *FsnotifyWatcher) Listen(rule *fsnotify.Rule, taskChan chan *fsnotify.Event) {
	go this.realTimeMonitGuard(rule, taskChan)
}

func (this *FsnotifyWatcher) realTimeMonit(rule *fsnotify.Rule, taskChan chan *fsnotify.Event) {
	watcher, err := fsnotify.NewRecursiveWatcher()
	if err != nil {
		this.logger.Fatalf("The error occured when create watcher: %s", err)
		return
	}
	defer watcher.Close()

	go watcher.NotifyFsEvent(rule, func(e *fsnotify.Event, err error) {
		if err != nil {
			this.logger.Errorf("The error occured during monitoring filesystem: %s", err)
			// GetQueuedCompletionPort: The specified network name is no longer available.
			if strings.Contains(err.Error(), "The specified network name is no longer available") {
				this.logger.Errorf("%s fsnotify need to be reset", rule.Biz)
				this.reset = true
			}
			return
		}
		this.logger.Infof("Catched filesystem event: %s %s", e.Op, e.Name)
		if e.Op == "CREATE" || e.Op == "WRITE" {
			e.Biz = rule.Biz
			e.RootPath = rule.RootPath
			taskChan <- e
		}
	})

	err = watcher.RecursiveAdd(rule)
	if err != nil {
		this.logger.Fatalf("The error occured when add the monitored directory: %s", err)
		return
	}

	<-rule.Done
	this.logger.Errorf("Trigger done channel, Biz: %s, Path: %s", rule.Biz, rule.MonitPath)
}

// 检测到目录突然没了，则得标记状态
// 目录突然又出现了，判断之前是否有有异常，如果没有，则不做处理，如果之前报错，则重新注册监听程序
func (this *FsnotifyWatcher) realTimeMonitGuard(rule *fsnotify.Rule, taskChan chan *fsnotify.Event) {
	rule.Done = make(chan struct{})
	go this.realTimeMonit(rule, taskChan)

	for {
		time.Sleep(20 * time.Second)
		// fmt.Println("Monitor Watch:", rule.MonitPath)
		// 以下操作有两个目的:
		// 1) 保证网络目录重新挂载上之后再进行重置监听的操作
		// 2) 只有去访问目录，才会触发网络目录访问不到的错，所以访问操作必不可少
		if _, err := os.Stat(rule.MonitPath); err != nil {
			this.logger.Errorf("Something wrong with %s, %s", rule.Biz, err)
			continue
		}
		if this.reset {
			this.reset = false
			this.logger.Warnf("Reset %s all watches.", rule.Biz)
			// 重置之前的监控
			close(rule.Done)
			this.logger.Warnf("Removes %s all watches.", rule.Biz)

			// 必须要重新生成新通道，否则会将新创建的协程也关闭了
			rule.Done = make(chan struct{})
			go this.realTimeMonit(rule, taskChan)
			this.logger.Warnf("Restart %s all watches.", rule.Biz)
		}
	}
}
