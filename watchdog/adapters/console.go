package watchdog

import (
	"fmt"
	"time"
)

type ConsoleAdapter struct {
	Name string
}

func (this *ConsoleAdapter) Handle(files []string) error {
	// getFileMeta
	// write the filename to stdout
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}
