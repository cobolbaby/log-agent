package fsnotify

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type Event struct {
	Name     string
	Op       string
	Biz      string
	RootPath string
	ModTime  time.Time
	IsDir    bool
}

type Rule struct {
	Biz             string
	RootPath        string
	MonitPath       string
	Patterns        string
	Ignores         string
	MaxNestingLevel uint
	DebounceTime    time.Duration
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

func (w *RecursiveWatcher) NotifyFsEvent(rule *Rule, cb func(e *Event, err error)) {
	for {
		select {
		case event := <-w.Events:
			// 优化事件触发的时机
			if event.Op&fsnotify.Create == fsnotify.Create {
				// 目录--Create事件必须监控，同时考虑到后期会为新建目录添加相应的触发器，所以还需回调
				// 文件--如果新建文件的父目录被监控了，Create事件就会被抛出，所以无需再次添加至监控列表
				// 如果将文件也添加至监控列表，则内存中需要维护一个大的map，出现内存持续飙升的问题
				if !CheckIfMatch(event.Name, rule) || CheckIfIgnore(event.Name, rule) {
					// fmt.Printf("%s ignore fs event: %s %s", rule.Biz, event.Op, event.Name)
					continue
				}
				fi, err := os.Stat(event.Name)
				if err != nil {
					cb(nil, err)
					continue
				}
				if fi.IsDir() {
					w.RecursiveAdd(&Rule{
						MonitPath: event.Name,
						Patterns:  rule.Patterns,
						Ignores:   rule.Ignores,
					})
				}
				cb(&Event{
					Op:   "CREATE",
					Name: event.Name,
				}, nil)
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// 目录--Write事件因文件创建/删除引起，所以无需再次添加监控列表，也无需回调
				// e.g. Windows下生成一个新文件会触发两个事件: 文件创建事件和目录写入事件
				// TODO:上述说法在直接拷贝目录的业务场景下不成立，后续还得继续支持该业务场景
				// 文件--Write事件因文件内容修改引起，所以无需再次添加监控列表
				if !CheckIfMatch(event.Name, rule) || CheckIfIgnore(event.Name, rule) {
					// fmt.Printf("%s ignore fs event: %s %s", rule.Biz, event.Op, event.Name)
					continue
				}
				cb(&Event{
					Op:   "WRITE",
					Name: event.Name,
				}, nil)
				continue
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				// 目录--Remove事件因目录删除引起，如果将其移出监控列表，可优化内存使用，但当前业务不涉及移除目录的操作
				// 文件--Remove事件因文件删除引起，而监控列表中仅保存了目录，所以没有什么好移除的
				if !CheckIfMatch(event.Name, rule) || CheckIfIgnore(event.Name, rule) {
					// fmt.Printf("%s ignore fs event: %s %s", rule.Biz, event.Op, event.Name)
					continue
				}
				// w.Remove(event.Name)
				cb(&Event{
					Op:   "REMOVE",
					Name: event.Name,
				}, nil)
				continue
			}
			if event.Op&fsnotify.Rename == fsnotify.Rename {
				// 目录--Rename事件与目录删除等同，如果将其移出监控列表，可优化内存使用，但当前业务不涉及移除目录的操作
				// 文件--Rename事件与文件删除等同，而监控列表中仅保存了目录，所以没有什么好移除的
				if !CheckIfMatch(event.Name, rule) || CheckIfIgnore(event.Name, rule) {
					// fmt.Printf("%s ignore fs event: %s %s", rule.Biz, event.Op, event.Name)
					continue
				}
				// w.Remove(event.Name)
				cb(&Event{
					Op:   "RENAME",
					Name: event.Name,
				}, nil)
				continue
			}
		case err := <-w.Errors:
			// fmt.Printf("w.Errors: %s", err)
			cb(nil, err)
		}
	}
}

func (w *RecursiveWatcher) RecursiveAdd(rule *Rule) error {
	fi, err := os.Stat(rule.MonitPath)
	if err != nil {
		return err
	}
	fmt.Println("Add Watch:", rule.MonitPath)
	w.Add(rule.MonitPath)
	if !fi.IsDir() {
		return nil
	}
	return WalkDir(rule, 1, func(e *Event) error {
		// 监控目录就能实时监听目录下的文件了, 所以没必要下那么多监听事件
		if !e.IsDir {
			return nil
		}
		fmt.Println("Add Watch:", e.Name)
		return w.Add(e.Name)
	})
}

func WalkDir(rule *Rule, level uint, fn func(e *Event) error) error {
	// ReadDir reads the directory named by dirname and returns a list of directory entries sorted by filename.
	// entries, err := ioutil.ReadDir(dir)
	// Ref: https://flaviocopes.com/go-list-files/
	f, err := os.Open(rule.MonitPath)
	if err != nil {
		return err
	}
	entries, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		subdir := filepath.Join(rule.MonitPath, entry.Name())
		// 非匹配项就不再遍历
		if !CheckIfMatch(subdir, rule) || CheckIfIgnore(subdir, rule) {
			continue
		}
		fn(&Event{
			Name:    subdir,
			ModTime: entry.ModTime(),
			IsDir:   entry.IsDir(),
			Op:      "LOAD",
		})
		// 支持设定目录监控的深度
		if entry.IsDir() && (rule.MaxNestingLevel == 0 || (rule.MaxNestingLevel != 0 && level < rule.MaxNestingLevel)) {
			r := new(Rule)
			*r = *rule
			r.MonitPath = subdir
			WalkDir(r, level+1, fn)
		}
	}
	return nil
}

func CheckIfMatch(path string, rule *Rule) bool {
	if rule.Patterns == ".*" || rule.Patterns == "" {
		return true
	}
	// 匹配规则中分隔符写法仅支持Linux风格
	return rule.Patterns != "" && regexp.MustCompile(rule.Patterns).MatchString(filepath.ToSlash(path))
}

func CheckIfIgnore(path string, rule *Rule) bool {
	if rule.Ignores == ".*" {
		return true
	}
	// 匹配规则中分隔符写法仅支持Linux风格
	return rule.Ignores != "" && regexp.MustCompile(rule.Ignores).MatchString(filepath.ToSlash(path))
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
