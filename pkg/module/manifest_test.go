package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValid(t *testing.T) {
	dir := t.TempDir()
	content := `{"name": "my-module", "version": "1.0.0", "exports": ["fn1", "fn2"]}`
	os.WriteFile(filepath.Join(dir, "pipe.json"), []byte(content), 0644)

	m, err := Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "my-module" {
		t.Errorf("name: expected my-module, got %s", m.Name)
	}
	if m.Version != "1.0.0" {
		t.Errorf("version: expected 1.0.0, got %s", m.Version)
	}
	if len(m.Exports) != 2 {
		t.Errorf("exports: expected 2, got %d", len(m.Exports))
	}
}

func TestParseMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Parse(dir)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pipe.json"), []byte("not json"), 0644)
	_, err := Parse(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEmptyName(t *testing.T) {
	m := &Manifest{Version: "1.0.0"}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateEmptyVersion(t *testing.T) {
	m := &Manifest{Name: "test"}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestValidateInvalidNameChars(t *testing.T) {
	m := &Manifest{Name: "Bad Name!", Version: "1.0.0"}
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error for invalid name chars")
	}
}

func TestValidateDependencies(t *testing.T) {
	m := &Manifest{
		Name:    "my-module",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"pipe-http": "^1.0.0",
			"sqlite":    "^0.8.0",
		},
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestInitModule(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-module")
	if err := InitModule(dir, "new-module"); err != nil {
		t.Fatal(err)
	}
	if !Exists(dir) {
		t.Fatal("expected pipe.json to exist")
	}
	m, err := Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "new-module" {
		t.Errorf("expected new-module, got %s", m.Name)
	}
	if m.Version != "0.1.0" {
		t.Errorf("expected 0.1.0, got %s", m.Version)
	}
	if _, err := os.Stat(filepath.Join(dir, "module.pipe")); err != nil {
		t.Fatal("expected module.pipe to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Fatal("expected README.md to exist")
	}
}

func TestInitModuleAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pipe.json"), []byte("{}"), 0644)
	err := InitModule(dir, "test")
	if err == nil {
		t.Fatal("expected error when pipe.json already exists")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if Exists(dir) {
		t.Fatal("should not exist yet")
	}
	os.WriteFile(filepath.Join(dir, "pipe.json"), []byte("{}"), 0644)
	if !Exists(dir) {
		t.Fatal("should exist now")
	}
}
