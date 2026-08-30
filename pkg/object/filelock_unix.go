//go:build !windows && !js

package object

import (
	"os"

	"golang.org/x/sys/unix"
)

// platformFileLock blocks until an exclusive advisory lock (flock(2), LOCK_EX)
// is held on f. Advisory locks are released automatically if the holding
// process dies without calling file_unlock, so a crashed process can't
// deadlock other processes out of the file permanently.
func platformFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

func platformFileUnlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
