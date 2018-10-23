package watchdog

import (
	"errors"
	"github.com/astaxie/beego/cache"
	"github.com/cobolbaby/log-agent/watchdog/handler"
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/hook"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"github.com/cobolbaby/log-agent/watchdog/watcher"
	"github.com/djherbis/times"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Watchdog struct {
	host     string
	logger   log.Logger
	watchers map[string]watcher.Watcher
	rules    map[string][]string
	adapters map[string][]handler.WatchdogHandler // 优先级队列
	cacheQ   []fsnotify.FileEvent
	hook     *hook.AdvanceHook
}

func NewWatchdog() *Watchdog {
	return &Watchdog{
		rules:    make(map[string][]string),
		watchers: make(map[string]watcher.Watcher),
		adapters: make(map[string][]handler.WatchdogHandler),
		hook:     hook.NewAdvanceHook(),
	}
}

func (this *Watchdog) SetHost(host string) *Watchdog {
	this.host = host
	return this
}

func (this *Watchdog) SetLogger(logger log.Logger) *Watchdog {
	this.logger = logger
	return this
}

func (this *Watchdog) SetWatcher(biz string, listener watcher.Watcher) *Watchdog {
	// this.watchers[biz] = listener

	return this
}

func (this *Watchdog) SetRules(biz string, rule string) *Watchdog {
	// 将rules按照分隔符拆分，合并当前规则
	ruleSlice := strings.Split(rule, ",")
	this.rules[biz] = append(this.rules[biz], ruleSlice...)
	return this
}

func (this *Watchdog) AddHandler(biz string, adapter ...handler.WatchdogHandler) *Watchdog {
	this.adapters[biz] = append(this.adapters[biz], adapter...)

	// 按照Priority排序
	adapters := this.adapters[biz]
	sort.SliceStable(adapters, func(i, j int) bool { return adapters[i].GetPriority() > adapters[j].GetPriority() })
	this.adapters[biz] = adapters

	return this
}

func (this *Watchdog) LoadPlugins(plugin hook.AdvancePlugin) *Watchdog {
	this.hook.Import(plugin)
	return this
}

func (this *Watchdog) Run() {
	// AutoCheck hook
	this.hook.Trigger("AutoCheck")
	// Init hook
	this.hook.Trigger("Init", this)

	// 支持同时配置多种业务的监控策略
	for biz, rules := range this.rules {
		aRule := &watcher.Rule{
			Biz:            biz,
			Rules:          rules,
			DelayQueueChan: make(chan fsnotify.FileEvent),
			Delay:          3 * time.Second,
			TaskQueueChan:  make(chan []fsnotify.FileEvent),
		}
		// go this.RegErrChan(aRule)
		go this.Listen(aRule)
		go this.TransferDebounce(aRule)
		// go this.Transfer(aRule)
		go this.Handle(aRule)
	}

	done := make(chan bool)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
}

func (this *Watchdog) Listen(rule *watcher.Rule) {
	// this.watchers[rule.Biz].Listen(rule)

	// 监听文件变化，则调用fsnotify
	if true {
		go watcher.NewFsnotifyWatcher().Listen(rule)
	}
	// 导入目录下原有文件，则调用fspolling
	if true {
		go watcher.NewFspollingWatcher().Listen(rule)
	}

	done := make(chan bool)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
}

func (this *Watchdog) TransferDebounce(rule *watcher.Rule) {
	timer := time.NewTimer(rule.Delay)
	var e fsnotify.FileEvent
	for {
		select {
		case e = <-rule.DelayQueueChan:
			this.cacheQ = append(this.cacheQ, e)
			timer.Reset(rule.Delay)
		case <-timer.C:
			if len(this.cacheQ) == 0 {
				break
			}
			rule.TaskQueueChan <- this.cacheQ
			this.cacheQ = []fsnotify.FileEvent{}
		}
	}
}

func (this *Watchdog) Transfer(rule *watcher.Rule) {
	var e fsnotify.FileEvent
	for {
		select {
		case e = <-rule.DelayQueueChan:
			rule.TaskQueueChan <- []fsnotify.FileEvent{e}
		}
	}
}

func (this *Watchdog) Handle(rule *watcher.Rule) {
	var e []fsnotify.FileEvent
	for {
		select {
		case e = <-rule.TaskQueueChan:
			this.handle(e)
		}
	}
}

func (this *Watchdog) handle(fileEvents []fsnotify.FileEvent) {
	this.adapterHandle(this.filterEvents(fileEvents))
}

func (this *Watchdog) filterEvents(fileEvents []fsnotify.FileEvent) []fsnotify.FileEvent {
	var list []fsnotify.FileEvent
	keys := make(map[string]bool)
	// 倒序，确保list中维护一个最新的事件列表
	for i := len(fileEvents) - 1; i >= 0; i-- {
		filename := fileEvents[i].Name
		if _, ok := keys[filename]; !ok {
			keys[filename] = true
			list = append(list, fileEvents[i])
		}
	}
	return list
}

func (this *Watchdog) getFileMeta(fileEvent fsnotify.FileEvent) (*handler.FileMeta, error) {
	fileInfo, err := os.Lstat(fileEvent.Name)
	if err != nil {
		return new(handler.FileMeta), err
	}
	if fileInfo.IsDir() {
		return new(handler.FileMeta), errors.New("[getFileMeta]仅处理文件，忽略目录")
	}

	// 获取文件目录
	// Ref: https://golang.org/pkg/path/filepath/#Split
	dirName, _ := filepath.Split(fileEvent.Name)

	// 获取文件相关时间，支持跨平台
	var fileCreateTime time.Time
	fileTime := times.Get(fileInfo)
	if fileTime.HasChangeTime() { // 非Win
		fileCreateTime = fileTime.ChangeTime()
	}
	if fileTime.HasBirthTime() { // Win
		fileCreateTime = fileTime.BirthTime()
	}

	return &handler.FileMeta{
		Filepath:   fileEvent.Name,  // 全路径
		Dirname:    dirName,         // 文件父目录
		Filename:   fileInfo.Name(), // 仅文件名
		Ext:        filepath.Ext(fileInfo.Name()),
		Size:       fileInfo.Size(),
		CreateTime: fileCreateTime,
		ModifyTime: fileInfo.ModTime(),
		LastOp:     fileEvent,
		Host:       this.host,
	}, nil
}

func (this *Watchdog) adapterHandle(files []fsnotify.FileEvent) {

	bm, err := cache.NewCache("file", `{"CachePath":"./.cache","FileSuffix":".txt","DirectoryLevel":2, "EmbedExpiry":0}`)
	if err != nil {
		this.logger.Error("[NewCache]%s", err)
		return
	}

	for _, file := range files {

		go func(file fsnotify.FileEvent) {
			// 获取file简要信息
			fileMeta, err := this.getFileMeta(file)
			if err != nil {
				// TODO:异常处理
				this.logger.Error("[getFileMeta]%s", err)
				return
			}

			// 支持Agent层级的清洗操作
			this.hook.Trigger("CheckFile", fileMeta)
			this.hook.Trigger("Transform", fileMeta)
			// ...
			// TODO:文件处理异常需要将该文件事件传送会DelayQueueChan

			failure := false
			// 考虑到失败回滚，采用串行更为便利
			for _, Adapter := range this.adapters[fileMeta.LastOp.Biz] {
				Adapter.SetLogger(this.logger)
				if err := Adapter.Handle(*fileMeta); err != nil {
					// TODO:失败重试
					this.logger.Error("File Handle Error: %s", err)
					failure = true
					break
				}
			}
			if failure {
				this.logger.Error("Need To Rollback File: %s", fileMeta.Filepath)
				this.adapterRollback(fileMeta)
				return
			}

			// 记录文件最新的md5值
			this.logger.Info("original file %s modtime : %s", file.Name, bm.Get(file.Name))
			bm.Put(file.Name, fileMeta.ModifyTime.String(), 0)
			this.logger.Info("changed file %s modtime : %s", file.Name, bm.Get(file.Name))
		}(file)
	}
}

func (this *Watchdog) adapterRollback(file *handler.FileMeta) {
	// 	var syncWg sync.WaitGroup
	// 	for _, Adapter := range this.adapters[file.LastOp.Biz] {
	// 		syncWg.Add(1)
	// 		go func(adapterhandler.WatchdogHandler) {
	// 			defer syncWg.Done()

	// 			go adapter.SetLogger(this.logger).Rollback(*file)
	// 		}(Adapter)
	// 	}
	// 	syncWg.Wait()

	// 	// TODO:将处理失败的事件传送至失败通道
}
