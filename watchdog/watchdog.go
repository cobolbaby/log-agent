package watchdog

import (
	"errors"
	. "github.com/cobolbaby/log-agent/watchdog/lib"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
}

type FileMeta struct {
	Filepath     string // 绝对路径
	Dirname      string
	Filename     string
	Size         int64
	Ext          string
	ModifyTime   time.Time
	UploadTime   time.Time
	ChunkData    []byte
	ChunkNo      uint32
	ChunkSize    uint64
	Compress     bool
	CompressSize int64
	Checksum     string
	LastOp       fsnotify.Event
}

type WatchdogAdapter interface {
	Handle(changeFile FileMeta) error
	SetLogger(logger Logger) WatchdogAdapter
}

type Watchdog struct {
	logger   Logger
	rules    []string
	adapters []WatchdogAdapter
	fsEventQ []fsnotify.Event
}

func Create() *Watchdog {
	this := &Watchdog{}
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
	this.adapters = append(this.adapters, adapter)
	return this
}

func (this *Watchdog) Run() {
	taskQueueChan := make(chan fsnotify.Event)
	go this.DebounceHandle(taskQueueChan, 3*time.Second)
	this.Listen(func(e fsnotify.Event) {
		taskQueueChan <- e
	})
}

func (this *Watchdog) Listen(cb func(e fsnotify.Event)) error {
	watcher, err := NewRecursiveWatcher()
	if err != nil {
		this.logger.Error("[NewRecursiveWatcher]%s", err)
		return err
	}
	defer watcher.Close()

	go watcher.NotifyFsEvent(cb)

	for _, rule := range this.rules {
		this.logger.Info("Listen Path: %s", rule)
		err := watcher.RecursiveAdd(rule)
		if err != nil {
			this.logger.Error("[RecursiveAdd]%s", err)
			return err
		}
	}

	done := make(chan bool)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
	return nil
}

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
			this.handle(this.fsEventQ)
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
	return this.adapterHandle(changeFileMeta)
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

	// Ref: https://golang.org/pkg/path/filepath/#Split
	dirName, fileName := filepath.Split(fileEvent.Name)
	dirNameAbs, err := filepath.Abs(dirName)
	if err != nil {
		return new(FileMeta), err
	}

	return &FileMeta{
		Filepath:   filepath.Join(dirNameAbs, fileName),
		Dirname:    dirName,
		Filename:   fileName,
		Ext:        filepath.Ext(fileName),
		Size:       fileInfo.Size(),
		ModifyTime: fileInfo.ModTime(),
		LastOp:     fileEvent,
	}, nil
}

func (this *Watchdog) adapterHandle(files []FileMeta) error {
	var wg sync.WaitGroup
	for _, fi := range files {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(fi FileMeta) {
			defer wg.Done()

			// TODO:分布式事务
			for _, Adapter := range this.adapters {
				Adapter.SetLogger(this.logger).Handle(fi)
			}

		}(fi)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	return nil
}
