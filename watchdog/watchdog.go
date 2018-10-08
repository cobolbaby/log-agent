package watchdog

import (
	"errors"
	. "github.com/cobolbaby/log-agent/watchdog/watchers"
	"github.com/djherbis/times"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Plugin interface {

} 

type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
}

type FileMeta struct {
	Filepath     string
	Pack         string
	Dirname      string
	Filename     string
	Size         int64 // 字节
	Ext          string
	CreateTime   time.Time
	ModifyTime   time.Time
	Content      []byte
	LastOp       fsnotify.Event
	BackUpTime   time.Time // 文件备份时间
	Checksum     string
	Compress     bool
	CompressSize int64
	Reference    string // 保留字段
	Host         string // 文件溯源
}

type fileChangeEvent struct {
	Name string
	Op	 string
}

type WatchdogAdapter interface {
	Handle(changeFile FileMeta) error
	SetLogger(logger Logger) WatchdogAdapter
	Rollback(changeFile FileMeta) error
}

type WatchdogListener interface {
	Listen() error
	SetLogger(logger Logger) WatchdogListener
}

type Watchdog struct {
	host     string
	logger   Logger
	rules    []string[] // 二维数组
	adapters []WatchdogAdapter
	fsEventQ []fsnotify.Event
	watcher  WatchdogListener
}

func NewWatchdog() *Watchdog {
	this := &Watchdog{}
	return this
}

func (this *Watchdog) SetHost(host string) *Watchdog {
	this.host = host
	return this
}

func (this *Watchdog) SetLogger(logger Logger) *Watchdog {
	this.logger = logger
	return this
}

func (this *Watchdog) SetRules(rule string) *Watchdog {
	// 将rules按照分隔符拆分，合并当前规则
	ruleSlice := strings.Split(rule, ",")
	this.rules = append(this.rules, ruleSlice...)
	return this
}

func (this *Watchdog) AddHandler(adapter WatchdogAdapter) *Watchdog {
	// TODO:实现优先级队列
	this.adapters = append(this.adapters, adapter)
	return this
}

func (this *Watchdog) pushHandleStack(channel, handler WatchdogAdapter) *Watchdog {
	// TODO:实现优先级队列
	// this.adapters = append(this.adapters, adapter)
	this.Handler[channel][] = {handler: handler, privilege: 6}
	return this
}

func (this *Watchdog) SetWatcher(watcher WatchdogAdapter)  *Watchdog{
	this.watcher =  watcher
	return this
}

func (this *Watchdog) SetPluginPath(pluginPath string) {
	this.pluginPath = pluginPath
	return this
}

// 加载具体的服务
func (this *Watchdog) LoadActivePlugins(plugin) {
	// this.plugins = append(this.plugins, "spi")
	if plugin.isActive()
		plugin.init(this)
	return this
}

func (this *Watchdog) Run() {
	taskQueueChan := make(chan fileChangeEvent)
	DelayQueueChan := make(chan )
	// 延迟处理通道
	// go this.DebounceHandle(taskQueueChan, 3*time.Second)
	// // TODO:策略模式
	// // TODO:支持监控目录的初次加载
	// this.Listen(func(e fsnotify.Event) {
	// 	taskQueueChan <- e
	// })

	go this.ListenV2(taskQueueChan)
	go this.DebounceV2(3*time.Second, taskQueueChan, DelayQueueChan)
	go this.HandleV2(DelayQueueChan)

	done := make(chan bool)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
}

func HandleV2(DelayQueueChan)  {
	for {
		select {
		case e = <-DelayQueueChan:
			this.handle(e)
		}
	}
}

func (this *Watchdog) ListenV2(taskQueueChan chan fileChangeEvent) error {
	// TODO:   循环rule
	rules := {
		val: this.rules.key.value
		key: key
	}
	
	
	this.watcher.Listen(taskQueueChan,rules)
}

// func (this *Watchdog) Listen(cb func(e fsnotify.Event)) error {
// 	watcher, err := NewRecursiveWatcher()
// 	if err != nil {
// 		this.logger.Error("[NewRecursiveWatcher]%s", err)
// 		return err
// 	}
// 	defer watcher.Close()

// 	go watcher.NotifyFsEvent(cb)

// 	for _, rule := range this.rules {
// 		this.logger.Info("Listen Path: %s", rule)
// 		err := watcher.RecursiveAdd(rule)
// 		if err != nil {
// 			this.logger.Error("[RecursiveAdd]%s", err)
// 			return err
// 		}
// 	}

// 	done := make(chan bool)
// 	// 如果done中还没放数据，那main挂起，直到放数据为止
// 	<-done
// 	return nil
// }

func (this *Watchdog) DebounceHandle(handleChan chan fsnotify.Event, interval time.Duration) {
	timer := time.NewTimer(interval)
	var e fsnotify.Event
	for {
		select {
		case e = <-handleChan:
			this.fsEventQ = append(this.fsEventQ, e)
			timer.Reset(interval)
		case <-timer.C:
			if len(this.fsEventQ) == 0 {
				break
			}
			// this.handle(this.fsEventQ)
			sss <- this.fsEvent
			this.fsEventQ = []fsnotify.Event{}
		}
	}
}

func (this *Watchdog) Handle(handleChan chan fsnotify.Event) {
	var e fsnotify.Event
	for {
		select {
		case e = <-handleChan:
			this.handle([]fsnotify.Event{e})
		}
	}
}

func (this *Watchdog) handle(fileEvents []fsnotify.Event) error {
	fileEvents = this.filterEvents(fileEvents)
	// 获取changeFiles的metadata
	changeFileMeta, err := this.getFileMeta(fileEvents)
	if err != nil {
		return err
	}
	// 保证数据的一致性
	this.adapterHandle(changeFileMeta, this.adapterRollback)
	return nil
}

func (this *Watchdog) filterEvents(fileEvents []fsnotify.Event) []fsnotify.Event {
	var list []fsnotify.Event
	keys := make(map[string]bool)
	// 倒序，确保list中维护一个最新的事件列表
	sort.SliceStable(fileEvents, func(i, j int) bool { return j < i })
	for _, entry := range fileEvents {
		if _, value := keys[entry.Name]; !value {
			keys[entry.Name] = true
			list = append(list, entry)
		}
	}
	return list
}

func (this *Watchdog) getFileMeta(eventQ []fsnotify.Event) ([]FileMeta, error) {
	var fileMetas []FileMeta
	for _, event := range eventQ {
		fileMeta, err := this.getOneFileMeta(event)
		if err != nil {
			return nil, err
		}
		fileMetas = append(fileMetas, *fileMeta)
	}
	return fileMetas, nil
}

func (this *Watchdog) getOneFileMeta(fileEvent fsnotify.Event) (*FileMeta, error) {
	fileInfo, err := os.Lstat(fileEvent.Name)
	if err != nil {
		return new(FileMeta), err
	}
	if fileInfo.IsDir() {
		return new(FileMeta), errors.New("[getOneFileMeta]仅处理文件，忽略目录")
	}

	// 获取文件目录
	// Ref: https://golang.org/pkg/path/filepath/#Split
	dirName, _ := filepath.Split(fileEvent.Name)
	// 获取文件相关时间，支持跨平台
	// fileTime, err := times.Stat(fileEvent.Name)
	// if err != nil {
	// 	return new(FileMeta), err
	// }
	// var fileCreateTime time.Time
	// if fileTime.HasChangeTime() { // 非Win
	// 	fileCreateTime = fileTime.ChangeTime()
	// }
	// if fileTime.HasBirthTime() { // Win
	// 	fileCreateTime = fileTime.BirthTime()
	// }
	fileTime := times.Get(fileInfo)

	// fileCreateTime, _ := time.Parse("2006-01-02 15:04:05-0700", "2018-09-28 08:15:22+0000")
	// TODO:矫正文件的创建时间
	fileCreateTime := fileTime.ChangeTime().Truncate(time.Millisecond).UTC()

	return &FileMeta{
		Filepath:   fileEvent.Name,
		Dirname:    dirName,
		Filename:   fileInfo.Name(),
		Ext:        filepath.Ext(fileInfo.Name()),
		Size:       fileInfo.Size(),
		CreateTime: fileCreateTime,
		ModifyTime: fileInfo.ModTime().Truncate(time.Millisecond).UTC(),
		LastOp:     fileEvent,
		Host:       this.host,
	}, nil
}

func (this *Watchdog) adapterHandle(files []FileMeta, cb func(file FileMeta)) {
	var wg sync.WaitGroup
	// TODO: pool
	for _, fi := range files {
		wg.Add(1)
		go func(file FileMeta) {
			defer wg.Done()

			failure := false
			for _, Adapter := range this.adapters {
				err := Adapter.SetLogger(this.logger).Handle(file)
				if err != nil {
					// TODO:失败重试
					failure = true
					break
				}
			}
			if failure {
				this.logger.Error("Need To Rollback File: %s", file.Filepath)
				cb(file)
			}
		}(fi)
	}
	wg.Wait()
}

func (this *Watchdog) adapterRollback(file FileMeta) {
	var syncWg sync.WaitGroup
	for _, Adapter := range this.adapters {
		syncWg.Add(1)
		go func(adapter WatchdogAdapter) {
			defer syncWg.Done()

			go adapter.SetLogger(this.logger).Rollback(file)
		}(Adapter)
	}
	syncWg.Wait()

	// TODO:将处理失败的事件传送至失败通道
}
