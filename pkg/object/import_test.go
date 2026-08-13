package object

import (
	"os"
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
