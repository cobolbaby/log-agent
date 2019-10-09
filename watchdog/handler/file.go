package handler

import (
	timeUtil "github.com/cobolbaby/log-agent/watchdog/lib/ctime"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"github.com/otiai10/copy"
	"path"
	"syscall"
	"time"
)

type FileAdapter struct {
	Name     string
	Config   *FileAdapterCfg
	logger   log.Logger
	Priority uint8
}

type FileAdapterCfg struct {
	DestRoot       string
	CustomPathFunc func(*FileMeta) string
	Name           string
}

func NewFileAdapter(Cfg *FileAdapterCfg) (WatchdogHandler, error) {
	name := "File"
	if Cfg.Name != "" {
		name = Cfg.Name
	}
	return &FileAdapter{
		Name:   name,
		Config: Cfg,
	}, nil
}

func (this *FileAdapter) SetLogger(logger log.Logger) {
	this.logger = logger
}

func (this *FileAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *FileAdapter) Handle(fi FileMeta) error {
	// 拷贝文件至目标目录
	var destPath string
	if this.Config.CustomPathFunc == nil {
		destPath = path.Join(this.Config.DestRoot, fi.SubDir, fi.Filename)
	} else {
		destPath = this.Config.CustomPathFunc(&fi)
	}
	// 若目标地址未设定，则不做操作
	if destPath == "" {
		return nil
	}
	if err := copy.Copy(fi.Filepath, destPath); err != nil {
		this.logger.Errorf("[FileAdapter] %s Failed to copy, %s", fi.Filepath, err)
		return err
	}
	// 要确保备份文件的创建时间不变
	if err := Chtimes(destPath, fi.CreateTime, fi.ModifyTime, fi.ModifyTime); err != nil {
		this.logger.Errorf("[FileAdapter] Failed to rsync the create time of %s, %s", fi.Filepath, err)
		return err
	}
	this.logger.Debugf("[FileAdapter] %s rsync to %s", fi.Filepath, destPath)
	return nil
}

func (this *FileAdapter) Rollback(fi FileMeta) error {
	return nil
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying filesystem may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func Chtimes(name string, ctime time.Time, atime time.Time, mtime time.Time) error {
	var utimes [3]syscall.Timespec
	utimes[0] = syscall.NsecToTimespec(atime.UnixNano())
	utimes[1] = syscall.NsecToTimespec(mtime.UnixNano())
	utimes[2] = syscall.NsecToTimespec(ctime.UnixNano())
	if e := timeUtil.UtimesNano(name, utimes[0:]); e != nil {
		return e
	}
	return nil
}
