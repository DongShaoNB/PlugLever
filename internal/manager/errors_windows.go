//go:build windows

package manager

import (
	"errors"
	"os"
	"syscall"
)

// Windows 系统调用中常见的文件占用与锁错误码
const (
	ERROR_SHARING_VIOLATION syscall.Errno = 32
	ERROR_LOCK_VIOLATION    syscall.Errno = 33
)

// isFileLockedOrPermission 判断错误是否由文件被其他进程占用（共享冲突）或权限不足引起
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
		case syscall.ERROR_ACCESS_DENIED, ERROR_SHARING_VIOLATION, ERROR_LOCK_VIOLATION:
			return true
		}
	}

	return false
}
