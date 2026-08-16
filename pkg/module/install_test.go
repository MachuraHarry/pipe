package module

import (
	"testing"
)

func TestLockfileRoundTrip(t *testing.T) {
	dir := t.TempDir()

	lock := &Lockfile{Modules: map[string]LockEntry{
		"sqlite": {
			Version:  "0.8.0",
			URL:      "https://example.com/modules/sqlite/module.pipe",
			Checksum: "abc123",
			Dependencies: map[string]string{
				"pipe-test": "^1.0.0",
			},
		},
	}}

	if err := WriteLockfile(dir, lock); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	got, err := ReadLockfile(dir)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}

	entry, ok := got.Modules["sqlite"]
	if !ok {
		t.Fatalf("expected sqlite entry in lockfile")
	}
	if entry.Version != "0.8.0" || entry.Checksum != "abc123" || entry.URL != "https://example.com/modules/sqlite/module.pipe" {
		t.Errorf("unexpected entry: %+v", entry)
	}
	if dep := entry.Dependencies["pipe-test"]; dep != "^1.0.0" {
		t.Errorf("expected dependency ^1.0.0, got %q", dep)
	}
}

func TestReadLockfileMissing(t *testing.T) {
	_, err := ReadLockfile(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing pipe.lock")
	}
}

func TestResolveDepsPrefersExistingLockfile(t *testing.T) {
	existing := &Lockfile{Modules: map[string]LockEntry{
		"sqlite": {
			Version:  "0.8.0",
			URL:      "https://example.com/sqlite/module.pipe",
			Checksum: "deadbeef",
		},
	}}

	lock := &Lockfile{Modules: make(map[string]LockEntry)}
	if err := resolveDeps(map[string]string{"sqlite": "^0.8.0"}, lock, nil, existing); err != nil {
		t.Fatalf("resolveDeps: %v", err)
	}

	entry, ok := lock.Modules["sqlite"]
	if !ok {
		t.Fatal("expected sqlite to be resolved from existing lockfile")
	}
	if entry.Version != "0.8.0" {
		t.Errorf("expected pinned version 0.8.0, got %q", entry.Version)
	}
	if entry.URL != "https://example.com/sqlite/module.pipe" {
		t.Errorf("expected pinned URL, got %q", entry.URL)
	}
}

func TestResolveDepsResolvesTransitiveFromExistingLockfile(t *testing.T) {
	existing := &Lockfile{Modules: map[string]LockEntry{
		"sqlite": {
			Version: "0.8.0",
			URL:     "https://example.com/sqlite/module.pipe",
			Dependencies: map[string]string{
				"pipe-test": "^1.0.0",
			},
		},
		"pipe-test": {
			Version: "1.1.0",
			URL:     "https://example.com/pipe-test/module.pipe",
		},
	}}

	lock := &Lockfile{Modules: make(map[string]LockEntry)}
	if err := resolveDeps(map[string]string{"sqlite": "^0.8.0"}, lock, nil, existing); err != nil {
		t.Fatalf("resolveDeps: %v", err)
	}

	if _, ok := lock.Modules["sqlite"]; !ok {
		t.Fatal("expected sqlite in lock")
	}
	if _, ok := lock.Modules["pipe-test"]; !ok {
		t.Fatal("expected transitive dependency pipe-test in lock")
	}
}
