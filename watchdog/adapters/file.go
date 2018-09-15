package watchdog

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
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}
