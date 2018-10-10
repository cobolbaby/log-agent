package watcher

type FspollingWatcher struct{}

func NewFspollingWatcher() *FspollingWatcher {
	return &FspollingWatcher{}
}