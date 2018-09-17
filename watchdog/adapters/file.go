package watchdog

import (
	"fmt"
	"time"
)

type FileAdapter struct {
	Name string
}

func (this *FileAdapter) Handle(files []string) error {
	// getFileMeta
	// mv
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}
