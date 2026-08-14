package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionRegexp(t *testing.T) {
	valid := []string{"1.0.0", "0.1.0", "10.20.30", "1.2.3-alpha", "1.0.0-beta.1.2", "v2.0.0"}
	invalid := []string{"1.0", "1", "latest", "1.2.3.4", "a.b.c", ""}
	for _, v := range valid {
		if !versionRegexp.MatchString(v) {
			t.Errorf("version %q should match", v)
		}
	}
	for _, v := range invalid {
		if versionRegexp.MatchString(v) {
			t.Errorf("version %q should not match", v)
		}
	}
}

func TestPublishValidatesManifest(t *testing.T) {
	dir := t.TempDir()
	if err := InitModule(dir, "pubtest"); err != nil {
		t.Fatal(err)
	}
	// Invalid version must be rejected before any network/gh interaction.
	m, err := Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.Version = "not-a-semver"
	if err := m.Write(dir); err != nil {
		t.Fatal(err)
	}
	if err := Publish(dir); err == nil {
		t.Fatal("expected error for invalid version, got nil")
	}
}

func TestPublishMissingModuleFile(t *testing.T) {
	dir := t.TempDir()
	if err := InitModule(dir, "pubtest"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "module.pipe")); err != nil {
		t.Fatal(err)
	}
	if err := Publish(dir); err == nil {
		t.Fatal("expected error for missing module.pipe, got nil")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0644)
	os.MkdirAll(filepath.Join(src, "sub"), 0755)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(src, "pipe.lock"), []byte("{}"), 0644)

	dst := filepath.Join(t.TempDir(), "out")
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(data) != "a" {
		t.Errorf("a.txt not copied correctly: %v %q", err, data)
	}
	if data, err := os.ReadFile(filepath.Join(dst, "sub", "b.txt")); err != nil || string(data) != "b" {
		t.Errorf("sub/b.txt not copied correctly: %v %q", err, data)
	}
	if _, err := os.Stat(filepath.Join(dst, "pipe.lock")); !os.IsNotExist(err) {
		t.Errorf("pipe.lock should not be copied, got %v", err)
	}
}
