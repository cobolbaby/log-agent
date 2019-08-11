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
	SOURCE_QUEUE_CAP         = 100              // 新消息上报通道容量
	CACHE_QUEUE_CAP          = 100              // 缓存处理通道容量
	TASK_QUEUE_CAP           = 2                // 待处理任务通道容量
	TASK_CONCURRENCY_CONTROL = 100              // 任务并发控制
	FS_POLL_INTERVAL         = 10 * time.Minute // 文件系统轮询时间间隔
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
	// 同时监控多种业务
	cacheQueueChan := make(chan *fsnotify.Event, CACHE_QUEUE_CAP)
	taskQueueChan := make(chan []*fsnotify.Event, TASK_QUEUE_CAP)
	for _, aRule := range this.rules {
		srcQueueChan := make(chan *fsnotify.Event, SOURCE_QUEUE_CAP)

		go this.listen(aRule, srcQueueChan)

		// 针对不同的业务可配置不同的延迟处理时间
		if aRule.DebounceTime > 0 {
			go this.debounce(aRule, srcQueueChan, cacheQueueChan)
		} else {
			go this.transfer(srcQueueChan, cacheQueueChan)
		}
	}
	// 采用协程池处理文件事件
	go this.transferBatch(200*time.Millisecond, cacheQueueChan, taskQueueChan)
	go this.handle(taskQueueChan)
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

func (this *Watchdog) listen(rule *fsnotify.Rule, destChan chan *fsnotify.Event) {
	// 导入目录下原有文件，则调用fspolling
	if InSlice(watcher.FS_POLL, this.watchers[rule.Biz]) {
		err := watcher.NewFspollingWatcher(FS_POLL_INTERVAL).SetLogger(this.Logger).Listen(rule, destChan)
		if err != nil {
			this.Logger.Error("[FspollingWatcher.Listen] %s", err)
		}
	}
	// 监听文件变化，则调用fsnotify
	if InSlice(watcher.FS_NOTIFY, this.watchers[rule.Biz]) {
		err := watcher.NewFsnotifyWatcher().SetLogger(this.Logger).Listen(rule, destChan)
		if err != nil {
			this.Logger.Error("[FsnotifyWatcher.Listen] %s", err)
		}
	}
}

func (this *Watchdog) debounce(rule *fsnotify.Rule, srcChan chan *fsnotify.Event, destChan chan *fsnotify.Event) {
	var debounceMap sync.Map
	for {
		select {
		case e := <-srcChan:
			eventChan, ok := debounceMap.Load(e.Name)
			if !ok {
				this.Logger.Debug("The debounce channel of %s is not hit", e.Name)
				// If not, add it to the debounce map
				eventChan := make(chan *fsnotify.Event)
				debounceMap.Store(e.Name, eventChan)
				this.Logger.Debug("Store %s to the debounce map", e.Name)

				// Start the debounce handler
				go this.debounceFsnotifyEvent(rule.DebounceTime, eventChan, func(event *fsnotify.Event) {
					this.Logger.Info("Debounce %s event for %s ms: %s %s", rule.Biz, rule.DebounceTime, event.Op, event.Name)
					debounceMap.Delete(event.Name)
					this.Logger.Debug("Delete %s in the debounce map", event.Name)
					destChan <- event
					return
				})

				eventChan <- e
				this.Logger.Debug("Publish %s to the debounce channel", e.Name)
			} else {
				// Publish the event to the channel of the debounce handler
				eventChan.(chan *fsnotify.Event) <- e
				this.Logger.Debug("Publish %s to the debounce channel", e.Name)
			}
		}
	}
}

func (this *Watchdog) debounceFsnotifyEvent(delay time.Duration, eventChan chan *fsnotify.Event, cb func(event *fsnotify.Event)) {
	// try to read from channel, block at most 5s.
	// if timeout, print time event and go on loop.
	// if read a message which is not the type we want(we want true, not false),
	// retry to read.
	timer := time.NewTimer(delay)
	var e *fsnotify.Event
	for {
		select {
		case e = <-eventChan:
			// timer may be not active, and fired
			if !timer.Stop() && len(timer.C) > 0 {
				<-timer.C //ctry to drain from the channel
			}
			timer.Reset(delay)
		case <-timer.C:
			// fmt.Println(time.Now(), ":timer expired")
			if e != nil {
				cb(e)
			}
			return
		}
	}
}

func (this *Watchdog) transferBatch(delay time.Duration, srcChan chan *fsnotify.Event, destChan chan []*fsnotify.Event) {
	timer := time.NewTicker(delay)
	var cacheQ []*fsnotify.Event
	for {
		select {
		case e := <-srcChan:
			cacheQ = append(cacheQ, e)
			if len(cacheQ) >= TASK_CONCURRENCY_CONTROL {
				destChan <- this.filterEvents(cacheQ)
				cacheQ = nil
			}
		case <-timer.C:
			if len(cacheQ) > 0 {
				destChan <- this.filterEvents(cacheQ)
				cacheQ = nil
			}
		}
	}
}

func (this *Watchdog) transfer(srcChan chan *fsnotify.Event, destChan chan *fsnotify.Event) {
	for {
		select {
		case e := <-srcChan:
			destChan <- e
		}
	}
}

func (this *Watchdog) handle(taskChan chan []*fsnotify.Event) {
	// 采用线程池的方式处理，有效节省处理大量协程时协程切换的开销
	pool := tunny.NewFunc(runtime.NumCPU(), func(payload interface{}) interface{} {

		this.fileProcessor(payload.(*fsnotify.Event))

		// 延时处理以降低系统IO
		// time.Sleep(100 * time.Millisecond)
		return nil
	})
	defer pool.Close()

	for {
		select {
		case tasks := <-taskChan:
			start := time.Now() // get current time

			var wg sync.WaitGroup
			wg.Add(len(tasks))
			for _, t := range tasks {
				go func(event *fsnotify.Event) {
					defer wg.Done()
					pool.Process(event)
				}(t)
			}
			wg.Wait()

			this.Logger.Info("Finish %d tasks in %s", len(tasks), time.Since(start))
		}
	}
}

func (this *Watchdog) filterEvents(fevents []*fsnotify.Event) []*fsnotify.Event {
	var l []*fsnotify.Event
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

func (this *Watchdog) GetFileMeta(fevent *fsnotify.Event) (*handler.FileMeta, error) {
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

func (this *Watchdog) fileProcessor(fevent *fsnotify.Event) {
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
		this.rollback(fileMeta)
		return
	}

	// 记录文件最新的md5值
	this.cache.Put(fevent.Name, fileMeta.ModifyTime.String(), 0)
	this.Logger.Debug("Cache key: %s, value: %s, timeout: 0", fevent.Name, this.cache.Get(fevent.Name))
}

func (this *Watchdog) rollback(file *handler.FileMeta) {
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
