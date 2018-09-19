package watchdog

import (
	"fmt"
	"io"
	"os"
	"time"
)

type FileAdapter struct {
	Name 	string
	Config 	FileAdapterCfg
}

type FileAdapterCfg struct {
}

func (this *FileAdapter) Handle(files []FileMeta) error {
	// getFileMeta
	// mv
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)
	for _, v := range files {
		fmt.Println(v)
	}
	return nil
}

func copyFile(srcFile string, dstFile string) (writen int64, err error) {
	f1, err := os.Open(srcFile)
	if err != nil {
		return nil, err
	}
	defer f1.Close()

	f2, err := os.OpenFile(dstFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer f2.Close()

	return io.Copy(f1, f2)
}


// // Copy recursively copies the file, directory or symbolic link at src
// // to dst. The destination must not exist. Symbolic links are not
// // followed.
// //
// // If the copy fails half way through, the destination might be left
// // partially written.
// func Copy(src, dst string) error {
// 	srcInfo, srcErr := os.Lstat(src)
// 	if srcErr != nil {
// 		return srcErr
// 	}
// 	_, dstErr := os.Lstat(dst)
// 	if dstErr == nil {
// 		// TODO(rog) add a flag to permit overwriting?
// 		return fmt.Errorf("will not overwrite %q", dst)
// 	}
// 	if !os.IsNotExist(dstErr) {
// 		return dstErr
// 	}
// 	switch mode := srcInfo.Mode(); mode & os.ModeType {
// 	case os.ModeSymlink:
// 		return copySymLink(src, dst)
// 	case os.ModeDir:
// 		return copyDir(src, dst, mode)
// 	case 0:
// 		return copyFile(src, dst, mode)
// 	default:
// 		return fmt.Errorf("cannot copy file with mode %v", mode)
// 	}
// }

// func copySymLink(src, dst string) error {
// 	target, err := os.Readlink(src)
// 	if err != nil {
// 		return err
// 	}
// 	return os.Symlink(target, dst)
// }

// func copyFile(src, dst string, mode os.FileMode) error {
// 	srcf, err := os.Open(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer srcf.Close()
// 	dstf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
// 	if err != nil {
// 		return err
// 	}
// 	defer dstf.Close()
// 	// Make the actual permissions match the source permissions
// 	// even in the presence of umask.
// 	if err := os.Chmod(dstf.Name(), mode.Perm()); err != nil {
// 		return err
// 	}
// 	if _, err := io.Copy(dstf, srcf); err != nil {
// 		return fmt.Errorf("cannot copy %q to %q: %v", src, dst, err)
// 	}
// 	return nil
// }

// func copyDir(src, dst string, mode os.FileMode) error {
// 	srcf, err := os.Open(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer srcf.Close()
// 	if mode&0500 == 0 {
// 		// The source directory doesn't have write permission,
// 		// so give the new directory write permission anyway
// 		// so that we have permission to create its contents.
// 		// We'll make the permissions match at the end.
// 		mode |= 0500
// 	}
// 	if err := os.Mkdir(dst, mode.Perm()); err != nil {
// 		return err
// 	}
// 	for {
// 		names, err := srcf.Readdirnames(100)
// 		for _, name := range names {
// 			if err := Copy(filepath.Join(src, name), filepath.Join(dst, name)); err != nil {
// 				return err
// 			}
// 		}
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return fmt.Errorf("error reading directory %q: %v", src, err)
// 		}
// 	}
// 	if err := os.Chmod(dst, mode.Perm()); err != nil {
// 		return err
// 	}
// 	return nil
// }