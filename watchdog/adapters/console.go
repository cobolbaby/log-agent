package adapters

import (
	"fmt"
)

type Console struct {
	Name string
}

func (this *Console) Handle(files []string) error {
	// getFileMeta
	// write the filename to stdout
	fmt.Println(this.Name)
	return nil
}
