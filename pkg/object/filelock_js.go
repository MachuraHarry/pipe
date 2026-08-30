//go:build js

package object

import "os"

// platformFileLock is a no-op under WebAssembly/js: there is no
// cross-process concept there (a single browser tab/worker owns the whole
// runtime), so file_lock/file_unlock always succeed immediately.
func platformFileLock(f *os.File) error {
	return nil
}

func platformFileUnlock(f *os.File) error {
	return nil
}
