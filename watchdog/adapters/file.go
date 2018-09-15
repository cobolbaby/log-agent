package adapters

import (
	"fmt"
)

type File struct {
	Name string
}

func (this *File) Handle(files []string) error {
	// getFileMeta
	// mv
	fmt.Println(this.Name)
	return nil
}
