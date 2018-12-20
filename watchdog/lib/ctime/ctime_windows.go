package ctime

import (
	"syscall"
)

func UtimesNano(path string, ts []syscall.Timespec) (err error) {
	if len(ts) != 3 {
		return syscall.EINVAL
	}
	pathp, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return e
	}
	h, e := syscall.CreateFile(pathp,
		syscall.FILE_WRITE_ATTRIBUTES, syscall.FILE_SHARE_WRITE, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if e != nil {
		return e
	}
	defer syscall.Close(h)
	a := syscall.NsecToFiletime(syscall.TimespecToNsec(ts[0]))
	w := syscall.NsecToFiletime(syscall.TimespecToNsec(ts[1]))
	c := syscall.NsecToFiletime(syscall.TimespecToNsec(ts[2]))
	return syscall.SetFileTime(h, &c, &a, &w)
}
