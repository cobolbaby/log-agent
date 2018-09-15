package watchdog

import (
	"fmt"
)

type Cassandra struct {
	Name string
}

func (this *Cassandra) Handle(files []string) error {
	// getConn
	// getFileMeta
	// UploadFile
	fmt.Println(">>>", this.Name)
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}
