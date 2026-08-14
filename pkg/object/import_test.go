package object

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModuleSpec(t *testing.T) {
	tests := []struct {
		input      string
		expectName string
		expectVer  string
	}{
		{"log-analyzer", "log-analyzer", ""},
		{"log-analyzer@1.0.0", "log-analyzer", "1.0.0"},
		{"log-analyzer@0.9.0", "log-analyzer", "0.9.0"},
		{"mylib", "mylib", ""},
		{"mylib@2", "mylib", "2"},
		{"./mylib.pipe", "./mylib.pipe", ""},
		{"../module/lib", "../module/lib", ""},
	}

	for _, tt := range tests {
		name, ver := parseModuleSpec(tt.input)
		if name != tt.expectName || ver != tt.expectVer {
			t.Errorf("parseModuleSpec(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.expectName, tt.expectVer)
		}
	}
}

func TestParseModuleSpecEdgeCases(t *testing.T) {
	// @ at the start should be ignored
	name, ver := parseModuleSpec("@latest")
	if name != "@latest" || ver != "" {
		t.Errorf("parseModuleSpec(@latest) = (%q, %q), want (@latest, '')", name, ver)
	}
}

func TestResolveImportAbsolutePath(t *testing.T) {
	tmp := t.TempDir()
	src := tmp + "/mathlib.pipe"
	if err := os.WriteFile(src, []byte("fn square x\n    x * x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	defer withProfile(testProfile("imp-abs", FSFull, true, false, true, nil))()

	path, content, err := ResolveImportFrom(src, "")
	if err != nil {
		t.Fatalf("absolute path import failed: %v", err)
	}
	if path != src {
		t.Errorf("path = %q, want %q", path, src)
	}
	if !strings.Contains(content, "fn square") {
		t.Errorf("content = %q, want to contain 'fn square'", content)
	}
}

func TestResolveImportAbsolutePathBlockedBySandbox(t *testing.T) {
	tmp := t.TempDir()
	secret := tmp + "/secret.pipe"
	if err := os.WriteFile(secret, []byte("print \"leak\""), 0644); err != nil {
		t.Fatal(err)
	}
	defer withProfile(testProfile("imp-locked", FSTempOnly, true, false, true, nil))()

	if _, _, err := ResolveImportFrom(secret, ""); err == nil {
		t.Fatal("expected import of file outside temp-only policy to be blocked, got nil")
	}
}

func TestResolveImportDirectoryInitPipe(t *testing.T) {
	tmp := t.TempDir()
	modDir := filepath.Join(tmp, "mylib")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	initPath := filepath.Join(modDir, "init.pipe")
	if err := os.WriteFile(initPath, []byte("export fn hello\n    \"hi\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIPE_PATH", tmp)
	sourceFile := filepath.Join(tmp, "main.pipe")

	// import "mylib/" (trailing slash)
	res, content, err := ResolveImportFrom("mylib/", sourceFile)
	if err != nil {
		t.Fatalf("directory import with slash failed: %v", err)
	}
	if res != initPath {
		t.Errorf("resolved path = %q, want %q", res, initPath)
	}
	if !strings.Contains(content, "fn hello") {
		t.Errorf("content = %q, want to contain 'fn hello'", content)
	}

	// import "mylib" (no slash, directory exists)
	res2, content2, err := ResolveImportFrom("mylib", sourceFile)
	if err != nil {
		t.Fatalf("directory import without slash failed: %v", err)
	}
	if res2 != initPath || content2 != content {
		t.Errorf("no-slash import = (%q, %q), want (%q, %q)", res2, content2, res, content)
	}
}

func TestResolveImportDirectoryWithoutInitPipe(t *testing.T) {
	tmp := t.TempDir()
	modDir := filepath.Join(tmp, "empty-mod")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIPE_PATH", tmp)
	sourceFile := filepath.Join(tmp, "main.pipe")

	if _, _, err := ResolveImportFrom("empty-mod/", sourceFile); err == nil {
		t.Fatal("expected directory without init.pipe to fail, got nil")
	}
}

func TestResolveImportRelative(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	utilsPath := filepath.Join(srcDir, "utils.pipe")
	if err := os.WriteFile(utilsPath, []byte("export fn square x\n    x * x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(srcDir, "main.pipe")

	// import "./utils.pipe"
	res, content, err := ResolveImportFrom("./utils.pipe", sourceFile)
	if err != nil {
		t.Fatalf("relative import ./ failed: %v", err)
	}
	if res != utilsPath {
		t.Errorf("resolved path = %q, want %q", res, utilsPath)
	}
	if !strings.Contains(content, "fn square") {
		t.Errorf("content = %q, want to contain 'fn square'", content)
	}

	// import "../other.pipe"
	otherPath := filepath.Join(tmp, "other.pipe")
	if err := os.WriteFile(otherPath, []byte("export fn greet\n    \"hi\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	res2, _, err := ResolveImportFrom("../other.pipe", sourceFile)
	if err != nil {
		t.Fatalf("relative import ../ failed: %v", err)
	}
	if res2 != otherPath {
		t.Errorf("resolved path = %q, want %q", res2, otherPath)
	}
}

func TestResolveImportRelativeDirectory(t *testing.T) {
	tmp := t.TempDir()
	subDir := filepath.Join(tmp, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	initPath := filepath.Join(subDir, "init.pipe")
	if err := os.WriteFile(initPath, []byte("export fn f\n    1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(tmp, "main.pipe")

	res, _, err := ResolveImportFrom("./pkg", sourceFile)
	if err != nil {
		t.Fatalf("relative directory import failed: %v", err)
	}
	if res != initPath {
		t.Errorf("resolved path = %q, want %q", res, initPath)
	}
}

func TestResolveImportMissingRelative(t *testing.T) {
	if _, _, err := ResolveImportFrom("./nope.pipe", "/tmp/does/not/exist/main.pipe"); err == nil {
		t.Fatal("expected missing relative import to fail, got nil")
	}
}
