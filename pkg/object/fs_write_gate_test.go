package object

import (
	"os"
	"path/filepath"
	"testing"
)

// withSandboxFSFlags mirrors withSandboxFlags (network_gate_test.go) for the
// filesystem-write dimension: it keeps ActiveProfile at "none" and governs
// writes via Sandbox.AllowFS, exactly the CLI `pipe --sandbox` path.
func withSandboxFSFlags(enabled, allowFS bool) func() {
	prevEnabled, prevAllowFS := Sandbox.Enabled, Sandbox.AllowFS
	prevProfile := ActiveProfile.Load()
	Sandbox.Enabled, Sandbox.AllowFS = enabled, allowFS
	ActiveProfile.Store(profileRegistry["none"])
	return func() {
		Sandbox.Enabled, Sandbox.AllowFS = prevEnabled, prevAllowFS
		ActiveProfile.Store(prevProfile)
	}
}

// The CLI --sandbox flag keeps ActiveProfile at "none" and governs
// filesystem writes via Sandbox.AllowFS. Every fs-write builtin must honor
// that flag, not just the registered-profile path: append_file, file_move,
// file_copy, make_dir, file_open (write modes) and write_file were found
// (round-7 audit, same bug class as the round-6 network gate) to check only
// the profile path and let a real write through under `pipe --sandbox`
// alone.
func TestFSWriteBuiltinsBlockedBySandboxFlag(t *testing.T) {
	dir := t.TempDir()
	defer withSandboxFSFlags(true, false)()

	cases := []struct {
		name string
		call func() Object
	}{
		{"append_file", func() Object {
			return bAppendFile(&String{Value: filepath.Join(dir, "a.txt")}, &String{Value: "x"})
		}},
		{"write_file", func() Object {
			return bWriteFile(&String{Value: filepath.Join(dir, "w.txt")}, &String{Value: "x"})
		}},
		{"file_delete", func() Object {
			return bFileDelete(&String{Value: filepath.Join(dir, "d.txt")})
		}},
		{"file_move", func() Object {
			return bFileMove(&String{Value: filepath.Join(dir, "src.txt")}, &String{Value: filepath.Join(dir, "dst.txt")})
		}},
		{"file_copy", func() Object {
			return bFileCopy(&String{Value: filepath.Join(dir, "src.txt")}, &String{Value: filepath.Join(dir, "dst.txt")})
		}},
		{"make_dir", func() Object {
			return bMakeDir(&String{Value: filepath.Join(dir, "newdir")})
		}},
		{"remove_dir", func() Object {
			return bRemoveDir(&String{Value: filepath.Join(dir, "newdir")})
		}},
		{"file_open write mode", func() Object {
			return bFileOpen(&String{Value: filepath.Join(dir, "fo.txt")}, &String{Value: "w"})
		}},
		{"file_open append mode", func() Object {
			return bFileOpen(&String{Value: filepath.Join(dir, "fo.txt")}, &String{Value: "a"})
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSandboxBlocked(t, tc.name, tc.call())
		})
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("blocked calls touched the filesystem, found: %v", entries)
	}
}

func TestFSWriteBuiltinsAllowedBySandboxFlag(t *testing.T) {
	dir := t.TempDir()
	defer withSandboxFSFlags(true, true)()

	target := filepath.Join(dir, "w.txt")
	if res := bWriteFile(&String{Value: target}, &String{Value: "hello"}); res.Type() == ERROR {
		t.Fatalf("write_file: expected success under AllowFS=true, got %s", res.Inspect())
	}
	data, e := os.ReadFile(target)
	if e != nil || string(data) != "hello" {
		t.Fatalf("write_file did not write the expected content: %v %q", e, data)
	}
}

// bFileOpen's read mode must remain ungated by the CLI --sandbox flag
// (only writes are restricted), matching read_file/read_lines/list_dir.
func TestFileOpenReadModeUnaffectedBySandboxFlag(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "r.txt")
	if err := os.WriteFile(target, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	defer withSandboxFSFlags(true, false)()

	res := bFileOpen(&String{Value: target}, &String{Value: "r"})
	if res.Type() == ERROR {
		t.Fatalf("file_open read mode: expected success under sandbox flag, got %s", res.Inspect())
	}
}
