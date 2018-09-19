package watchdog

import (
	"strings"
	"sync"
	"errors"
	"path/filepath"
)

const (
	FILE_MAX_SIZE = 16 * 1024 * 1024 // 16M
)

type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
}

type FileMeta struct {
	Filepath		string
	Dirname       	string
	Filename      	string
	Size          	uint64
	Ext      	  	string
	ModifyTime    	uint32
	UploadTime    	uint32
	ChunkData     	[]byte
	ChunkNo		 	uint64
	ChunkSize	  	uint64
	Compress      	bool
	CompressSize  	uint64
	Checksum      	string
}

type FileHandler interface {
	Handle(changeFiles []FileMeta) error
	SetLogger(logger Logger)
}

type Watchdog struct {
	logger   Logger
	rules    []string
	adapters []FileHandler
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

func (this *Watchdog) AddHandler(adapter FileHandler) *Watchdog {
	this.adapters = append(this.adapters, adapter)
	return this
}

func (this *Watchdog) Run() {
	this.listen(func(changeFiles []string) {
		
		// TODO:考虑做一个缓冲队列，然后分批次处理

		if len(changeFiles) > 0 {
			this.handle(changeFiles)
		}
		// ...
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
	changeFilesMeta, err := this.getFileMeta(changeFiles)
	if err != nil {
		return err
	}
	return this.adapterHandle(changeFilesMeta)
}

func (this *Watchdog) getFilesMeta(files []string) ([]FileMeta, error) {
	filesMeta := []FileMeta
	for _, fi := range files {
		fileMeta, err := this.getFileMeta(fi)
		if err != nil {
			return nil, err
		}
		filesMeta = append(filesMeta, fileMeta)
	}
	return filesMeta, nil
}

func (this *Watchdog) getFileMeta(file string) (FileMeta, error) {
	fileInfo, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return nil, errors.New("[getFileMeta]仅处理文件，忽略目录")
	}

	// TODO:针对超大文件执行过滤操作
	if fileInfo.Size() > FILE_MAX_SIZE {
		return nil, errors.New("[getFileMeta]仅处理小于16M的文件")
	}

	dirName, fileName := filepath.Split(file)
	dirName = filepath.Abs(dirName)
	return &FileMeta{
		Filepath:       filepath.Join(dirName, fileName),
		Dirname:		dirName,
		Filename:      	fileName,
		Ext:			filepath.Ext(fileName),
		Size:          	fileInfo.Size(),
		ModifyTime:    	fileInfo.ModTime(),
	}, nil
}

func (this *Watchdog) adapterHandle(files []FileMeta) error {
	var wg sync.WaitGroup
	for _, Adapter := range this.adapters {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(handler FileHandler) {
			defer wg.Done()
			handler.SetLogger(this.logger).Handle(files)
		}(Adapter)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	return nil
}