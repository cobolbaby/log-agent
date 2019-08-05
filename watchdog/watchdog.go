package watchdog

import (
	"github.com/cobolbaby/log-agent/watchdog/handler"
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/hook"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"github.com/cobolbaby/log-agent/watchdog/watcher"
	"github.com/Jeffail/tunny"
	"github.com/astaxie/beego/cache"
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
)

type Watchdog struct {
	host     string
	Logger   *log.LogMgr
	watchers map[string][]string
	rules    map[string]*fsnotify.Rule
	adapters map[string][]handler.WatchdogHandler // 优先级队列
	hook     *hook.AdvanceHook
	cache    cache.Cache
}

func NewWatchdog() *Watchdog {
	bm, _ := cache.NewCache("file", `{"CachePath":"./.cache","FileSuffix":".txt","DirectoryLevel":"2", "EmbedExpiry":"0"}`)

	return &Watchdog{
		watchers: make(map[string][]string),
		rules:    make(map[string]*fsnotify.Rule),
		adapters: make(map[string][]handler.WatchdogHandler),
		hook:     hook.NewAdvanceHook(),
		cache:    bm,
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
	this.watchers[biz] = strategy
	return this
}

func (this *Watchdog) SetDefaultWatchStrategy(strategy ...string) *Watchdog {
	for _, biz := range this.hook.GetPlugins() {
		this.SetWatchStrategy(biz.Name(), strategy)
	}
	return this
}

func (this *Watchdog) SetRules(biz string, rule *fsnotify.Rule) *Watchdog {
	this.rules[biz] = rule
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
			case handler.KAFKA:
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
	// this.SetDefaultWatchStrategy(watcher.FS_NOTIFY, watcher.FS_POLL)
	this.SetDefaultHandler(handler.Console)
	// 插件配置自检
	if err := this.hook.Listen("AutoCheck", this); err != nil {
		this.Logger.Fatal("AutoCheck hook throw exception: %s", err)
	}
	// 挂载插件初始化配置
	if err := this.hook.Listen("AutoInit", this); err != nil {
		this.Logger.Fatal("AutoInit hook throw exception: %s", err)
	}
	// 挂载插件自定义配置
	if err := this.hook.Listen("Mount", this); err != nil {
		this.Logger.Fatal("Mount hook throw exception: %s", err)
	}
	// 同时监控多种业务，并且针对不同的业务可配置不同的延迟处理时间
	TaskQueueChan := make(chan []*fsnotify.FileEvent, TaskQueueChanCap)
	DelayQueueChan := make(chan *fsnotify.FileEvent, DelayQueueChanCap)
	for _, aRule := range this.rules {
		go this.Listen(aRule, DelayQueueChan)
		// 通过延长Debounce时间来降低与其他进程产生竞争的概率
		go this.TransferDebounce(aRule.DebounceTime, DelayQueueChan, TaskQueueChan)
		// if aRule.DebounceTime > 0 {
		// 	go this.TransferDebounce(aRule.DebounceTime, DelayQueueChan, TaskQueueChan)
		// } else {
		// 	go this.Transfer(DelayQueueChan, TaskQueueChan)
		// }
	}
	// 采用协程池处理文件事件
	go this.Handle(TaskQueueChan)
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

func (this *Watchdog) Listen(rule *fsnotify.Rule, taskChan chan *fsnotify.FileEvent) {
	// 导入目录下原有文件，则调用fspolling
	if InSlice(watcher.FS_POLL, this.watchers[rule.Biz]) {
		err := watcher.NewFspollingWatcher(FsLoopScanInterval).SetLogger(this.Logger).Listen(rule, taskChan)
		if err != nil {
			this.Logger.Error("[FspollingWatcher.Listen] %s", err)
		}
	}
	// 监听文件变化，则调用fsnotify
	if InSlice(watcher.FS_NOTIFY, this.watchers[rule.Biz]) {
		err := watcher.NewFsnotifyWatcher().SetLogger(this.Logger).Listen(rule, taskChan)
		if err != nil {
			this.Logger.Error("[FsnotifyWatcher.Listen] %s", err)
		}
	}
}

func (this *Watchdog) TransferDebounce(delay time.Duration, srcChan chan *fsnotify.FileEvent, distChan chan []*fsnotify.FileEvent) {
	timer := time.NewTicker(delay)
	var cacheQ []*fsnotify.FileEvent
	var e *fsnotify.FileEvent

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

func (this *Watchdog) Transfer(srcChan chan *fsnotify.FileEvent, distChan chan []*fsnotify.FileEvent) {
	var e *fsnotify.FileEvent
	for {
		select {
		case e = <-srcChan:
			distChan <- []*fsnotify.FileEvent{e}
		}
	}
}

func (this *Watchdog) Handle(taskChan chan []*fsnotify.FileEvent) {
	var e []*fsnotify.FileEvent

	// 采用线程池的方式处理，有效节省处理大量协程时协程切换的开销
	pool := tunny.NewFunc(runtime.NumCPU(), func(payload interface{}) interface{} {

		this.fileProcessor(payload.(*fsnotify.FileEvent))

		// 延时处理以降低系统IO
		// time.Sleep(100 * time.Millisecond)
		return nil
	})
	defer pool.Close()

	for {
		select {
		case e = <-taskChan:
			this.adapterHandle(e, pool)
		}
	}
}

func (this *Watchdog) filterEvents(fevents []*fsnotify.FileEvent) []*fsnotify.FileEvent {
	var l []*fsnotify.FileEvent
	keys := make(map[string]bool)
	// 倒序，确保l中维护一个最新的事件列表
	for i := len(fevents) - 1; i >= 0; i-- {
		filename := fevents[i].Name
		if _, ok := keys[filename]; !ok {
			keys[filename] = true
			l = append(l, fevents[i])
		}
	}
	return l
}

func (this *Watchdog) GetFileMeta(fevent *fsnotify.FileEvent) (*handler.FileMeta, error) {
	fi, err := os.Lstat(fevent.Name)
	if err != nil {
		return &handler.FileMeta{}, err
	}
	if fi.IsDir() {
		return &handler.FileMeta{}, nil
	}

	// 文件目录，支持跨平台
	dirName := filepath.Dir(fevent.Name)
	// filepath.Clean 自动转化目录分隔符，如 "C:/dev/workspace" => "C:\\dev\\workspace"
	rootDirName := filepath.Clean(fevent.RootPath)
	var pathSeparator string
	if os.IsPathSeparator('\\') {
		pathSeparator = "\\"
	} else {
		pathSeparator = "/"
	}
	subDirName := filepath.ToSlash(strings.Trim(strings.Replace(dirName, rootDirName, "", 1), pathSeparator))

	// 文件创建时间，支持跨平台
	var fileCreateTime time.Time
	fileTime := times.Get(fi)
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
		Filepath:   fevent.Name,
		SubDir:     subDirName,
		Filename:   fi.Name(),
		Ext:        strings.ToLower(filepath.Ext(fi.Name())),
		Size:       fi.Size(),
		CreateTime: fileCreateTime,
		ModifyTime: fi.ModTime(),
		LastOp:     fevent,
		Host:       this.host,
		FolderTime: folderCreateTime,
	}, nil
}

func (this *Watchdog) fileProcessor(fevent *fsnotify.FileEvent) {
	// 获取file简要信息
	fileMeta, err := this.GetFileMeta(fevent)
	if err != nil {
		// FindFirstFile D:\\I1000_testlog\\HP\\Matterhorn\\K2786401B\\NULL.txt: The system cannot find the file specified.
		this.Logger.Warn("Fail to get origin file: %s", err)
		// 如果是文件被删除了, 该咋办?
		if err := this.hook.Listen("Handle404Error", this, fileMeta, fevent); err != nil {
			this.Logger.Warn("Handle404Error hook throw exception: %s", err)
			return
		}
	}
	if fileMeta.Filepath == "" {
		this.Logger.Warn("The fileMeta is an empty struct, please check the event: %s %s", fevent.Op, fevent.Name)
		return
	}

	// 支持Agent层级的清洗操作
	if err := this.hook.Listen("CheckFile", this, fileMeta); err != nil {
		this.Logger.Warn("CheckFile hook throw exception: %s", err)
		// 如果报文件不完整，那稍后应该还会有写入事件产生，所以暂不做处理
		return
	}
	this.hook.Listen("Transform", this, fileMeta)

	failure := false
	// 考虑到失败回滚，采用串行更为便利
	for _, Adapter := range this.adapters[fileMeta.LastOp.Biz] {
		Adapter.SetLogger(this.Logger)
		if err := Adapter.Handle(*fileMeta); err != nil {
			this.Logger.Error("Adapter.Handle throw exception: %s", err)
			failure = true
			break
		}
	}
	if failure {
		this.Logger.Error("Need to rollback file: %s", fileMeta.Filepath)
		// 文件处理异常时需要将该文件事件传送至异常处理通道
		this.adapterRollback(fileMeta)
		return
	}

	// 记录文件最新的md5值
	this.cache.Put(fevent.Name, fileMeta.ModifyTime.String(), 0)
	this.Logger.Debug("Cache key: %s, value: %s, timeout: 0", fevent.Name, this.cache.Get(fevent.Name))
}

func (this *Watchdog) adapterHandle(files []*fsnotify.FileEvent, pool *tunny.Pool) {
	startTime := time.Now() // get current time

	var wg sync.WaitGroup
	wg.Add(len(files))
	for _, file := range files {
		go func(e *fsnotify.FileEvent) {
			defer wg.Done()
			pool.Process(e)
		}(file)
	}
	wg.Wait()

	this.Logger.Info("Finish %d tasks in %s", len(files), time.Since(startTime))
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

	// TODO:将处理失败的事件传送至失败通道
}
