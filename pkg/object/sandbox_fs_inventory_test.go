package object

// Sandbox FS inventory: every file-read/write builtin audited against the
// temp-only redirect. The rule under a temp-only profile is that paths outside
// the sandbox temp dir are redirected into it: writes land in the temp copy
// (the outside file must remain untouched) and reads cannot leak outside
// content (the redirected path does not exist). This closes the audit item
// "File-I/O builtin inventory against the temp-only redirect" from the
// sandbox-audit reports.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempSandboxPath(dir, name string) string {
	return filepath.Join(dir, ".pipe_sandbox", name)
}

// writeOutside creates a file outside the sandbox temp dir and returns its path.
func writeOutside(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func assertOutsideUntouched(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("outside file vanished: %v", err)
	}
	if string(data) != want {
		t.Fatalf("outside file was modified: %q, want %q", data, want)
	}
}

func assertSandboxedContent(t *testing.T, dir, name, want string) {
	t.Helper()
	p := tempSandboxPath(dir, name)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("sandboxed copy %s not found: %v", p, err)
	}
	if string(data) != want {
		t.Fatalf("sandboxed copy content %q, want %q", data, want)
	}
}

func assertError(t *testing.T, res Object, what string) {
	t.Helper()
	if res.Type() != ERROR {
		t.Fatalf("%s: expected error, got: %s", what, res.Inspect())
	}
}

// ---- write builtins redirect into the temp dir ----

func TestTempOnlyWriteBuiltinsRedirect(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	// write_file
	outside := writeOutside(t, dir, "wf.txt", "original")
	res := bWriteFile(&String{Value: outside}, &String{Value: "sneaky"})
	if res.Type() == ERROR {
		t.Fatalf("write_file redirected write must not error: %s", res.Inspect())
	}
	assertOutsideUntouched(t, outside, "original")
	assertSandboxedContent(t, dir, "wf.txt", "sneaky")

	// append_file (redirect model: sandbox starts empty, so a redirect of an
	// outside path appends to a fresh sandbox copy)
	outside = writeOutside(t, dir, "af.txt", "original")
	res = bAppendFile(&String{Value: outside}, &String{Value: "+more"})
	if res.Type() == ERROR {
		t.Fatalf("append_file redirected write must not error: %s", res.Inspect())
	}
	assertOutsideUntouched(t, outside, "original")
	assertSandboxedContent(t, dir, "af.txt", "+more")

	// file_open in every write mode redirects
	for _, mode := range []string{"w", "a", "rw", "rw+"} {
		outside = writeOutside(t, dir, "open-"+mode+".txt", "original")
		res = bFileOpen(&String{Value: outside}, &String{Value: mode})
		if res.Type() == ERROR {
			t.Fatalf("file_open mode %q must not error: %s", mode, res.Inspect())
		}
		handle, _ := res.(*Integer)
		bFileClose(handle)
		assertOutsideUntouched(t, outside, "original")
		if _, err := os.Stat(tempSandboxPath(dir, "open-"+mode+".txt")); err != nil {
			t.Fatalf("file_open mode %q did not create sandboxed copy (err=%v)", mode, err)
		}
	}
}

func TestTempOnlyFileDeleteRedirects(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := writeOutside(t, dir, "fd.txt", "keep")
	sandboxed := tempSandboxPath(dir, "fd.txt")
	if err := os.MkdirAll(filepath.Dir(sandboxed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sandboxed, []byte("discard"), 0644); err != nil {
		t.Fatal(err)
	}

	res := bFileDelete(&String{Value: outside})
	if res.Type() == ERROR {
		t.Fatalf("file_delete redirected delete must not error: %s", res.Inspect())
	}
	assertOutsideUntouched(t, outside, "keep")
	if _, err := os.Stat(sandboxed); !os.IsNotExist(err) {
		t.Fatalf("sandboxed copy was not deleted (err=%v)", err)
	}
}

func TestTempOnlyRemoveDirRedirects(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := filepath.Join(dir, "rmdir")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	sandboxed := tempSandboxPath(dir, "rmdir")
	if err := os.MkdirAll(sandboxed, 0755); err != nil {
		t.Fatal(err)
	}

	res := bRemoveDir(&String{Value: outside})
	if res.Type() == ERROR {
		t.Fatalf("remove_dir redirected delete must not error: %s", res.Inspect())
	}
	if _, err := os.Stat(filepath.Join(outside, "keep.txt")); err != nil {
		t.Fatalf("outside dir was removed: %v", err)
	}
	if _, err := os.Stat(sandboxed); !os.IsNotExist(err) {
		t.Fatalf("sandboxed dir was not removed (err=%v)", err)
	}
}

func TestTempOnlyFileCopyRedirects(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	// Source lives inside the sandbox (visible); destination is an outside
	// path that must be redirected into the temp dir without touching outside.
	srcInSandbox := tempSandboxPath(dir, "fc-src.txt")
	if err := os.MkdirAll(filepath.Dir(srcInSandbox), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcInSandbox, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	outsideDst := writeOutside(t, dir, "fc-dst.txt", "original")

	res := bFileCopy(&String{Value: srcInSandbox}, &String{Value: outsideDst})
	if res.Type() == ERROR {
		t.Fatalf("file_copy redirected copy must not error: %s", res.Inspect())
	}
	assertOutsideUntouched(t, outsideDst, "original")
	assertSandboxedContent(t, dir, "fc-dst.txt", "data")
}

func TestTempOnlyFileMoveRedirects(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	srcInSandbox := tempSandboxPath(dir, "fm-src.txt")
	if err := os.MkdirAll(filepath.Dir(srcInSandbox), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcInSandbox, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	outsideDst := writeOutside(t, dir, "fm-dst.txt", "original")

	res := bFileMove(&String{Value: srcInSandbox}, &String{Value: outsideDst})
	if res.Type() == ERROR {
		t.Fatalf("file_move redirected move must not error: %s", res.Inspect())
	}
	assertOutsideUntouched(t, outsideDst, "original")
	assertSandboxedContent(t, dir, "fm-dst.txt", "data")
	if _, err := os.Stat(srcInSandbox); !os.IsNotExist(err) {
		t.Fatalf("sandboxed source was not moved away (err=%v)", err)
	}
}

func TestTempOnlyMakeDirRedirects(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := filepath.Join(dir, "newdir")
	res := bMakeDir(&String{Value: outside})
	if res.Type() == ERROR {
		t.Fatalf("make_dir redirected mkdir must not error: %s", res.Inspect())
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside dir was created (err=%v)", err)
	}
	if fi, err := os.Stat(tempSandboxPath(dir, "newdir")); err != nil || !fi.IsDir() {
		t.Fatalf("sandboxed dir was not created (err=%v)", err)
	}
}

// ---- read builtins cannot leak outside content ----

func TestTempOnlyReadBuiltinsNoLeak(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := writeOutside(t, dir, "secret.txt", "SECRET")

	// read_file
	res := bReadFile(&String{Value: outside})
	assertError(t, res, "read_file of outside file")
	if strings.Contains(res.Inspect(), "SECRET") {
		t.Fatalf("read_file leaked outside content: %s", res.Inspect())
	}

	// read_lines
	res = bReadLines(&String{Value: outside})
	assertError(t, res, "read_lines of outside file")

	// file_size / file_type on the redirected (non-existent) path error
	assertError(t, bFileSize(&String{Value: outside}), "file_size of outside file")
	assertError(t, bFileType(&String{Value: outside}), "file_type of outside file")

	// file_open in read mode cannot open the outside file
	res = bFileOpen(&String{Value: outside}, &String{Value: "r"})
	assertError(t, res, "file_open(r) of outside file")
}

func TestTempOnlyFileExistsRedirectsToTemp(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := writeOutside(t, dir, "ex.txt", "SECRET")

	// The outside file must not be visible as existing; the redirected path
	// does not exist, so file_exists reports false.
	res := bFileExists(&String{Value: outside})
	if b, ok := res.(*Boolean); !ok || b.Value {
		t.Fatalf("file_exists of outside file must be false, got: %s", res.Inspect())
	}

	// A sandboxed file of the same name IS visible (redirect semantics).
	assertSandboxedContent := tempSandboxPath(dir, "ex.txt")
	if err := os.WriteFile(assertSandboxedContent, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	res = bFileExists(&String{Value: outside})
	if b, ok := res.(*Boolean); !ok || !b.Value {
		t.Fatalf("file_exists of sandboxed copy must be true, got: %s", res.Inspect())
	}
}

func TestTempOnlyListDirRedirects(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := filepath.Join(dir, "leakdir")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("SECRET"), 0644); err != nil {
		t.Fatal(err)
	}

	res := bListDir(&String{Value: outside})
	if res.Type() == ERROR {
		return // redirected path missing -> error is acceptable and safe
	}
	lst, ok := res.(*List)
	if !ok {
		t.Fatalf("list_dir returned non-list: %s", res.Inspect())
	}
	for _, e := range lst.Elements {
		if s, ok := e.(*String); ok && s.Value == "secret.txt" {
			t.Fatalf("list_dir leaked outside directory listing")
		}
	}
}

// ---- isolated (fs: none) profiles block all file builtins ----

func TestIsolatedBlocksFileBuiltins(t *testing.T) {
	p := NewSandboxProfile("iso-fs-inv")
	p.FSAccess = FSNone
	p.PathPolicy = &defaultPathPolicy{access: FSNone}
	defer withProfile(p)()

	assertError(t, bReadFile(&String{Value: "/etc/hostname"}), "read_file under isolated")
	assertError(t, bReadLines(&String{Value: "/etc/hostname"}), "read_lines under isolated")
	assertError(t, bWriteFile(&String{Value: "/tmp/x.txt"}, &String{Value: "x"}), "write_file under isolated")
	assertError(t, bAppendFile(&String{Value: "/tmp/x.txt"}, &String{Value: "x"}), "append_file under isolated")
	assertError(t, bMakeDir(&String{Value: "/tmp/iso-dir"}), "make_dir under isolated")
	assertError(t, bFileOpen(&String{Value: "/tmp/x.txt"}, &String{Value: "w"}), "file_open under isolated")
	assertError(t, bFileCopy(&String{Value: "/a"}, &String{Value: "/b"}), "file_copy under isolated")
	assertError(t, bFileMove(&String{Value: "/a"}, &String{Value: "/b"}), "file_move under isolated")
}
