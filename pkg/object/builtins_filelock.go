package object

import (
	"fmt"
	"os"
)

// FileLock is an OS-level advisory exclusive lock on a file, held for the
// lifetime between file_lock and file_unlock. Unlike write_file/read_file,
// which only touch file *contents*, this coordinates access *across
// processes* — e.g. two separate `pipe` processes (a long-running bot loop
// and a web dashboard) sharing one SQLite-on-disk file that has no
// cross-process locking of its own.
type FileLock struct {
	Path string
	f    *os.File
}

func (fl *FileLock) Type() ObjectType { return "FILE_LOCK" }
func (fl *FileLock) Inspect() string  { return fmt.Sprintf("file-lock:%s", fl.Path) }

// file_lock(path) blocks until an exclusive OS-level lock on path is held,
// creating the file if it doesn't exist, and returns a lock handle. Two
// processes calling file_lock on the same path serialize: the second call
// blocks until the first calls file_unlock (or exits, which releases OS
// locks automatically). Gated by the same FS-write sandbox policy as
// write_file, since it creates/opens a file for writing.
func bFileLock(args ...Object) Object {
	if len(args) != 1 {
		return err("file_lock expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_lock: path must be a string")
	}
	path, cerr := checkFSWriteAccess("file_lock", p.Value)
	if cerr != nil {
		return cerr
	}
	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr != nil {
		return err("file_lock: " + openErr.Error())
	}
	if lockErr := platformFileLock(f); lockErr != nil {
		f.Close()
		return err("file_lock: " + lockErr.Error())
	}
	return &FileLock{Path: p.Value, f: f}
}

// file_unlock(lock) releases a lock acquired by file_lock.
func bFileUnlock(args ...Object) Object {
	if len(args) != 1 {
		return err("file_unlock expects 1 argument (a file_lock handle)")
	}
	fl, ok := args[0].(*FileLock)
	if !ok {
		return err("file_unlock: argument must be a file_lock handle")
	}
	if unlockErr := platformFileUnlock(fl.f); unlockErr != nil {
		fl.f.Close()
		return err("file_unlock: " + unlockErr.Error())
	}
	if closeErr := fl.f.Close(); closeErr != nil {
		return err("file_unlock: " + closeErr.Error())
	}
	return NILOBJ
}
