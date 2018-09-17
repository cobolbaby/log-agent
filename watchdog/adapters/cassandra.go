package watchdog

import (
	"fmt"
	"time"
)

type CassandraAdapter struct {
	Name string
}

func (this *CassandraAdapter) Handle(files []string) error {
	// getConn
	// getFileMeta
	// UploadFile
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}
