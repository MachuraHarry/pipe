package object

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileLockUnlockRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roundtrip.lock")

	lock := bFileLock(&String{Value: path})
	if lock.Type() == ERROR {
		t.Fatalf("file_lock returned error: %s", lock.Inspect())
	}
	if _, ok := lock.(*FileLock); !ok {
		t.Fatalf("file_lock returned %T, want *FileLock", lock)
	}

	if res := bFileUnlock(lock); res.Type() == ERROR {
		t.Fatalf("file_unlock returned error: %s", res.Inspect())
	}
}

func TestFileUnlockRejectsWrongType(t *testing.T) {
	if res := bFileUnlock(&String{Value: "not a lock"}); res.Type() != ERROR {
		t.Fatal("file_unlock on a non-lock value must return an error")
	}
}

// TestFileLockSerializesAcrossHandles is the actual point of file_lock: two
// independent open file handles on the same path (standing in for two
// separate `pipe` processes, e.g. a bot loop and a web dashboard, both
// touching one on-disk database file with no locking of its own) must not
// both believe they hold the lock at once. The second file_lock call must
// block until the first process's file_unlock releases it.
func TestFileLockSerializesAcrossHandles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serialize.lock")

	first := bFileLock(&String{Value: path})
	if first.Type() == ERROR {
		t.Fatalf("first file_lock returned error: %s", first.Inspect())
	}

	acquired := make(chan Object, 1)
	go func() {
		acquired <- bFileLock(&String{Value: path})
	}()

	select {
	case <-acquired:
		t.Fatal("second file_lock acquired the lock while the first handle still held it")
	case <-time.After(150 * time.Millisecond):
		// Expected: still blocked.
	}

	if res := bFileUnlock(first); res.Type() == ERROR {
		t.Fatalf("file_unlock (first) returned error: %s", res.Inspect())
	}

	select {
	case second := <-acquired:
		if second.Type() == ERROR {
			t.Fatalf("second file_lock returned error after release: %s", second.Inspect())
		}
		if res := bFileUnlock(second); res.Type() == ERROR {
			t.Fatalf("file_unlock (second) returned error: %s", res.Inspect())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second file_lock never acquired the lock after the first was released")
	}
}
