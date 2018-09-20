package watchdog

import (
	"errors"
	"os"
	"path/filepath"
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
	Filepath     string
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
}

type WatchdogAdapter interface {
	Handle(changeFiles []FileMeta) error
	SetLogger(logger Logger) WatchdogAdapter
}

type Watchdog struct {
	logger   Logger
	rules    []string
	adapters []WatchdogAdapter
}

func Create() *Watchdog {
	this := new(Watchdog)
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
	this.listen(func(changeFiles []string) {

		// TODO:考虑做一个缓冲队列，然后分批次处理

		if len(changeFiles) > 0 {
			this.handle(changeFiles)
		}
		// TODO:文件读取后的搬移、删除、保留
	})
}

func (this *Watchdog) listen(callback func(queue []string)) error {
	watcher, err := NewRecursiveWatcher()
	if err != nil {
		this.logger.Error("[NewRecursiveWatcher]", err)
		return err
	}
	defer watcher.Close()

	// ...
	done := make(chan bool)
	go watcher.RegCallback(callback)
	for _, rule := range this.rules {
		this.logger.Info("Listen: ", rule)
		err := watcher.RecursiveAdd(rule)
		if err != nil {
			this.logger.Error("[RecursiveAdd]", err)
			return err
		}
	}
	<-done

	return nil
}

func (this *Watchdog) handle(changeFiles []string) error {
	// 获取changeFiles的metadata
	changeFileMetas, err := this.getFileMetas(changeFiles)
	if err != nil {
		return err
	}
	return this.adapterHandle(changeFileMetas)
}

func (this *Watchdog) getFileMetas(files []string) ([]FileMeta, error) {
	var fileMetas []FileMeta
	for _, fi := range files {
		fileMeta, err := this.getOneFileMeta(fi)
		if err != nil {
			return nil, err
		}
		fileMetas = append(fileMetas, *fileMeta)
	}
	return fileMetas, nil
}

func (this *Watchdog) getOneFileMeta(file string) (*FileMeta, error) {
	fileInfo, err := os.Lstat(file)
	if err != nil {
		return new(FileMeta), err
	}
	if fileInfo.IsDir() {
		return new(FileMeta), errors.New("[getOneFileMeta]仅处理文件，忽略目录")
	}

	dirName, fileName := filepath.Split(file)
	dirName, err = filepath.Abs(dirName)
	if err != nil {
		return new(FileMeta), err
	}

	return &FileMeta{
		Filepath:   filepath.Join(dirName, fileName),
		Dirname:    dirName,
		Filename:   fileName,
		Ext:        filepath.Ext(fileName),
		Size:       fileInfo.Size(),
		ModifyTime: fileInfo.ModTime(),
	}, nil
}

func (this *Watchdog) adapterHandle(files []FileMeta) error {
	var wg sync.WaitGroup
	for _, Adapter := range this.adapters {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(apdater WatchdogAdapter) {
			defer wg.Done()
			apdater.SetLogger(this.logger).Handle(files)
			// apdater.Handle(files)
		}(Adapter)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	return nil
}
