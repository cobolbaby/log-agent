package watchdog

import (
	"dc-agent-go/watchdog/handler"
	"dc-agent-go/watchdog/lib/fsnotify"
	"dc-agent-go/watchdog/lib/hook"
	"dc-agent-go/watchdog/lib/log"
	"dc-agent-go/watchdog/watcher"
	"github.com/Jeffail/tunny"
	"github.com/astaxie/beego/cache"
	"github.com/bcicen/grmon/agent"
	"github.com/djherbis/times"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DelayQueueChanCap  = 100              // 延迟处理通道长度
	CacheQueueMaxSize  = 100              // 一次性处理的最大任务数
	TaskQueueChanCap   = 1                // 待处理任务通道长度
	FsLoopScanInterval = 30 * time.Minute // 文件系统轮询时间间隔
	DebounceTime       = 3 * time.Second  // 文件系统事件延迟处理时间
)

type Watchdog struct {
	host          string
	Logger        *log.LogMgr
	watchStrategy map[string][]string
	rules         map[string][]string
	adapters      map[string][]handler.WatchdogHandler // 优先级队列
	hook          *hook.AdvanceHook
	cache         cache.Cache
}

func NewWatchdog() *Watchdog {
	bm, _ := cache.NewCache("file", `{"CachePath":"./.cache","FileSuffix":".txt","DirectoryLevel":2, "EmbedExpiry":0}`)

	return &Watchdog{
		watchStrategy: make(map[string][]string),
		rules:         make(map[string][]string),
		adapters:      make(map[string][]handler.WatchdogHandler),
		hook:          hook.NewAdvanceHook(),
		cache:         bm,
	}
}

func (this *Watchdog) SetHost(hostname string) *Watchdog {
	this.host = hostname
	return this
}

func (this *Watchdog) SetLogger(logger *log.LogMgr) *Watchdog {
	this.Logger = logger
	return this
}

func (this *Watchdog) SetWatchStrategy(biz string, strategy []string) *Watchdog {
	this.watchStrategy[biz] = strategy
	return this
}

func (this *Watchdog) SetDefaultWatchStrategy(strategy ...string) *Watchdog {
	for _, biz := range this.hook.GetPlugins() {
		this.SetWatchStrategy(biz.Name(), strategy)
	}
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
	tmp := this.adapters[biz]
	sort.SliceStable(tmp, func(i, j int) bool { return tmp[i].GetPriority() > tmp[j].GetPriority() })
	this.adapters[biz] = tmp

	return this
}

// TODO:用于移除默认的操作
func (this *Watchdog) RemoveHandler(biz string, adapterName ...string) *Watchdog {
	return this
}

func (this *Watchdog) SetDefaultHandler(adapterNames ...string) *Watchdog {

	for _, biz := range this.hook.GetPlugins() {
		for _, adapter := range adapterNames {
			switch adapter {
			case handler.Console:
				ConsoleAdapter, _ := handler.NewConsoleAdapter()
				this.AddHandler(biz.Name(), ConsoleAdapter)
			case handler.Cassandra:
			case handler.File:
			case handler.RabbitMQ:
			default:
			}
		}
	}
	return this
}

func (this *Watchdog) LoadPlugins(plugins []hook.AdvancePlugin) *Watchdog {
	this.hook.Import(plugins...)
	return this
}

func (this *Watchdog) Run() {
	// 设置默认选项
	this.SetDefaultWatchStrategy(watcher.Fsnotify, watcher.Fspolling)
	this.SetDefaultHandler(handler.Console)

	// 插件配置自检
	if err := this.hook.Trigger("AutoCheck", this); err != nil {
		this.Logger.Fatal("AutoCheck hook return error, %s", err)
	}
	// 挂载插件初始化配置
	if err := this.hook.Trigger("AutoInit", this); err != nil {
		this.Logger.Fatal("AutoInit hook return error, %s", err)
	}
	// 挂载插件自定义配置
	if err := this.hook.Trigger("Mount", this); err != nil {
		this.Logger.Fatal("Mount hook return error, %s", err)
	}

	DelayQueueChan := make(chan fsnotify.FileEvent, DelayQueueChanCap)
	TaskQueueChan := make(chan []fsnotify.FileEvent, TaskQueueChanCap)
	// 支持同时配置多种业务监控策略
	for biz, rules := range this.rules {
		aRule := &watcher.Rule{
			Biz:   biz,
			Rules: rules,
		}
		go this.Listen(aRule, DelayQueueChan)
	}

	// bug: The process cannot access the file because it is being used by another process.
	// 通过延长Debounce时间来降低与其他进程产生竞争的概率
	go this.TransferDebounce(DelayQueueChan, TaskQueueChan, DebounceTime)

	go this.Handle(TaskQueueChan)

	// 启动程序监控
	grmon.Start()
	// TODO:推送心跳信息
}

// InSlice checks given string in string slice or not.
func InSlice(v string, sl []string) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

func (this *Watchdog) Listen(rule *watcher.Rule, taskChan chan fsnotify.FileEvent) {
	// 导入目录下原有文件，则调用fspolling
	if InSlice(watcher.Fspolling, this.watchStrategy[rule.Biz]) {
		err := watcher.NewFspollingWatcher(FsLoopScanInterval).SetLogger(this.Logger).Listen(rule, taskChan)
		if err != nil {
			this.Logger.Error("[FspollingWatcher] %s", err)
		}
	}
	// 监听文件变化，则调用fsnotify
	if InSlice(watcher.Fsnotify, this.watchStrategy[rule.Biz]) {
		err := watcher.NewFsnotifyWatcher().SetLogger(this.Logger).Listen(rule, taskChan)
		if err != nil {
			this.Logger.Error("[FsnotifyWatcher] %s", err)
		}
	}
}

func (this *Watchdog) TransferDebounce(srcChan chan fsnotify.FileEvent, distChan chan []fsnotify.FileEvent, delay time.Duration) {
	timer := time.NewTicker(delay)
	var cacheQ []fsnotify.FileEvent
	var e fsnotify.FileEvent

	// 实现带优先级的Channel
	for {
		select {
		case e = <-srcChan:
			cacheQ = append(cacheQ, e)
			if len(cacheQ) >= CacheQueueMaxSize {
				distChan <- this.filterEvents(cacheQ)
				cacheQ = nil
			}
		case <-timer.C:
			if len(cacheQ) > 0 {
				distChan <- this.filterEvents(cacheQ)
				cacheQ = nil
			}
		}
	}
}

func (this *Watchdog) Transfer(srcChan chan fsnotify.FileEvent, distChan chan []fsnotify.FileEvent) {
	var e fsnotify.FileEvent
	for {
		select {
		case e = <-srcChan:
			distChan <- []fsnotify.FileEvent{e}
		}
	}
}

func (this *Watchdog) Handle(taskChan chan []fsnotify.FileEvent) {
	var e []fsnotify.FileEvent
	for {
		select {
		case e = <-taskChan:
			this.adapterHandle(e)
		}
	}
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
		return nil, err
	}
	if fileInfo.IsDir() {
		// this.hook.Trigger("ProcessDirChange", this, fileInfo)
		return nil, nil
	}

	// 文件目录，支持跨平台
	dirName := filepath.Dir(fileEvent.Name)
	rootDirName := filepath.Clean(fileEvent.MonitorDir)
	var pathSeparator string
	if os.IsPathSeparator('\\') {
		pathSeparator = "\\"
	} else {
		pathSeparator = "/"
	}
	subDirName := filepath.ToSlash(strings.Trim(strings.Replace(dirName, rootDirName, "", 1), pathSeparator))

	// 文件创建时间，支持跨平台
	var fileCreateTime time.Time
	fileTime := times.Get(fileInfo)
	if fileTime.HasChangeTime() { // 非Win
		fileCreateTime = fileTime.ChangeTime()
	}
	if fileTime.HasBirthTime() { // Win
		fileCreateTime = fileTime.BirthTime()
	}

	// [新增]文件夹的创建时间，支持跨平台
	dirInfo, err := os.Lstat(dirName)
	if err != nil {
		return nil, err
	}
	var folderCreateTime time.Time
	folderTime := times.Get(dirInfo)
	if folderTime.HasChangeTime() { // 非Win
		folderCreateTime = folderTime.ChangeTime()
	}
	if folderTime.HasBirthTime() { // Win
		folderCreateTime = folderTime.BirthTime()
	}

	return &handler.FileMeta{
		Filepath:   fileEvent.Name,
		SubDir:     subDirName,
		Filename:   fileInfo.Name(),
		Ext:        strings.ToLower(filepath.Ext(fileInfo.Name())),
		Size:       fileInfo.Size(),
		CreateTime: fileCreateTime,
		ModifyTime: fileInfo.ModTime(),
		LastOp:     fileEvent,
		Host:       this.host,
		FolderTime: folderCreateTime,
	}, nil
}

func (this *Watchdog) fileProcessor(file fsnotify.FileEvent) {

	// 获取file简要信息
	fileMeta, err := this.getFileMeta(file)
	if err != nil {
		/*
			e.g.
			1) FindFirstFile D:\\I1000_testlog\\HP\\Matterhorn\\K2786401B\\NULL.txt: The system cannot find the file specified.
		*/
		this.Logger.Warn("[getFileMeta] %s %s", err, file)
		return
	}
	if fileMeta == nil {
		return
	}

	// 支持Agent层级的清洗操作
	this.hook.Trigger("CheckFile", this, fileMeta)
	this.hook.Trigger("Transform", this, fileMeta)
	// TODO:文件处理异常时需要将该文件事件传送至异常处理通道

	failure := false
	// 考虑到失败回滚，采用串行更为便利
	for _, Adapter := range this.adapters[fileMeta.LastOp.Biz] {
		Adapter.SetLogger(this.Logger)
		if err := Adapter.Handle(*fileMeta); err != nil {
			this.Logger.Error("[fileProcessor] File Handle Error: %s", err)
			failure = true
			break
		}
	}
	if failure {
		this.Logger.Error("[fileProcessor] Need To Rollback File: %s", fileMeta.Filepath)
		// TODO:文件处理异常时需要将该文件事件传送至异常处理通道
		this.adapterRollback(fileMeta)
		return
	}

	// 记录文件最新的md5值
	this.cache.Put(file.Name, fileMeta.ModifyTime.String(), 0)
	this.Logger.Debug("[Cache] %s %s", file.Name, this.cache.Get(file.Name))
}

func (this *Watchdog) adapterHandle(files []fsnotify.FileEvent) {
	numCPUs := runtime.NumCPU()
	numTask := len(files)

	if numTask < numCPUs {
		for _, file := range files {
			this.Logger.Info("Process %s", file.Name)
			go this.fileProcessor(file)
		}
		return
	}

	// 采用线程池的方式处理，有效节省处理大量协程时协程切换的开销
	pool := tunny.NewFunc(numCPUs, func(payload interface{}) interface{} {

		this.fileProcessor(payload.(fsnotify.FileEvent))

		// 延时处理以降低系统IO
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	defer pool.Close()

	startTime := time.Now() // get current time

	var wg sync.WaitGroup
	wg.Add(numTask)
	for _, file := range files {
		go func(input fsnotify.FileEvent) {
			defer wg.Done()
			pool.Process(input)
		}(file)
	}
	wg.Wait()

	this.Logger.Info("Finish %d tasks in %s", numTask, time.Since(startTime))
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
