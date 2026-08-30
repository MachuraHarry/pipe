//go:build windows

package object

import (
	"os"

	"golang.org/x/sys/windows"
)

// platformFileLock blocks until an exclusive lock is held on f via
// LockFileEx. A zero-valued OVERLAPPED with no LOCKFILE_FAIL_IMMEDIATELY flag
// makes this call synchronous/blocking even without FILE_FLAG_OVERLAPPED.
func platformFileLock(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, ol)
}

func platformFileUnlock(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}
