package ctime

import (
	"syscall"
)

func UtimesNano(path string, ts []syscall.Timespec) (err error) {
	return nil
}
