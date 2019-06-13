package fsnotify

import (
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type FileEvent struct {
	Biz        string
	ModTime    time.Time
	Op         string
	Name       string
	MonitorDir string
}

type RecursiveWatcher struct {
	*fsnotify.Watcher
}

func NewRecursiveWatcher() (*RecursiveWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &RecursiveWatcher{watcher}, nil
}

func (w *RecursiveWatcher) NotifyFsEvent(monitorDir string, cb func(err error, e FileEvent)) {
	for {
		select {
		case event := <-w.Events:
			// 优化事件触发的时机
			if event.Op&fsnotify.Create == fsnotify.Create {
				// 目录--Create事件必须监控，同时考虑到后期会为新建目录添加相应的触发器，所以还需回调
				// 文件--如果新建文件的父目录被监控了，Create事件就会被抛出，所以无需再次添加至监控列表
				// 如果将文件也添加至监控列表，则内存中需要维护一个大的map，出现内存持续飙升的问题
				fi, err := os.Stat(event.Name)
				if err != nil {
					cb(err, FileEvent{})
					continue
				}
				if fi.IsDir() {
					w.RecursiveAdd(event.Name, ".*")
				}
				cb(nil, FileEvent{
					Op:         "Create",
					Name:       event.Name,
					MonitorDir: monitorDir,
				})
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// 目录--Write事件因文件创建/删除引起，所以无需再次添加监控列表，也无需回调
				// e.g. Windows下生成一个新文件会触发两个事件: 文件创建事件和目录写入事件
				// 文件--Write事件因文件内容修改引起，所以无需再次添加监控列表
				fi, err := os.Stat(event.Name)
				if err != nil {
					cb(err, FileEvent{})
					continue
				}
				if fi.IsDir() {
					continue
				}
				cb(nil, FileEvent{
					Op:         "Write",
					Name:       event.Name,
					MonitorDir: monitorDir,
				})
				continue
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				// 目录--Remove事件因目录删除引起，如果将其移出监控列表，可优化内存使用，但当前业务不涉及移除目录的操作
				// 文件--Remove事件因文件删除引起，而监控列表中仅保存了目录，所以没有什么好移除的
				// w.Remove(event.Name)
				cb(nil, FileEvent{
					Op:         "Remove",
					Name:       event.Name,
					MonitorDir: monitorDir,
				})
				continue
			}
			if event.Op&fsnotify.Rename == fsnotify.Rename {
				// 目录--Rename事件与目录删除等同，如果将其移出监控列表，可优化内存使用，但当前业务不涉及移除目录的操作
				// 文件--Rename事件与文件删除等同，而监控列表中仅保存了目录，所以没有什么好移除的
				// w.Remove(event.Name)
				cb(nil, FileEvent{
					Op:         "Rename",
					Name:       event.Name,
					MonitorDir: monitorDir,
				})
				continue
			}
		case err := <-w.Errors:
			cb(err, FileEvent{})
		}
	}
}

func (w *RecursiveWatcher) RecursiveAdd(name string, exp string) error {
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		if err := w.Add(name); err != nil {
			return err
		}
		return nil
	}
	var re *regexp.Regexp
	if exp != ".*" && exp != "" {
		re = regexp.MustCompile(exp)
	}
	return filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		// 非匹配项不实时监控
		if re != nil && !re.MatchString(filepath.ToSlash(path)) {
			// 匹配规则中分隔符写法仅支持Linux风格
			return nil
		}
		return w.Add(path)
	})
}

/*
// filteredSearchOfDirectoryTree Walks down a directory tree looking for
// files that match the pattern: re. If a file is found print it out and
// add it to the files list for later user.
func filteredSearchOfDirectoryTree(re *regexp.Regexp, dir string) error {
	// Just a demo, this is how we capture the files that match
	// the pattern.
	files := []string{}

	// Function variable that can be used to filter
	// files based on the pattern.
	// Note that it uses re internally to filter.
	// Also note that it populates the files variable with
	// the files that matches the pattern.
	walk := func(fn string, fi os.FileInfo, err error) error {
		if re.MatchString(fn) == false {
			return nil
		}
		if fi.IsDir() {
			fmt.Println(fn + string(os.PathSeparator))
		} else {
			fmt.Println(fn)
			files = append(files, fn)
		}
		return nil
	}
	filepath.Walk(dir, walk)
	fmt.Printf("Found %[1]d files.\n", len(files))
	return nil
}
*/

func (w *RecursiveWatcher) RecursiveRemove(name string) error {
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		if err := w.Remove(name); err != nil {
			return err
		}
		return nil
	}
	return filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := w.Remove(path); err != nil {
			return err
		}
		return nil
	})
}
