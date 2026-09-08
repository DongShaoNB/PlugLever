//go:build !windows

package manager

import (
	"errors"
	"os"
	"syscall"
)

// isFileLockedOrPermission 判断错误是否由文件被其他进程占用或权限不足引起（非 Windows 平台）
func isFileLockedOrPermission(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}

	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.EACCES, syscall.EPERM, syscall.ETXTBSY:
			return true
		}
	}

	return false
}
