package adapters

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
	fmt.Println(this.Name)
	return nil
}
