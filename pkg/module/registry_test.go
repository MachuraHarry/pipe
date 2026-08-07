package module

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/object"
)

func TestGenerateRegistry(t *testing.T) {
	dir := t.TempDir()

	// Create module with pipe.json
	modDir := filepath.Join(dir, "my-module")
	os.MkdirAll(modDir, 0755)
	m := Manifest{
		Name:        "my-module",
		Version:     "1.2.3",
		Description: "A test module",
		Exports:     []string{"fn1", "fn2"},
	}
	m.Write(modDir)

	// Create legacy module without pipe.json (simulate with fake registry)
	legacyDir := filepath.Join(dir, "old-module")
	os.MkdirAll(legacyDir, 0755)
	os.WriteFile(filepath.Join(legacyDir, "module.pipe"), []byte("export fn old_fn x: x"), 0644)

	baseURL := "https://example.com/repo/master"

	if err := GenerateRegistry(dir, baseURL); err != nil {
		t.Fatal(err)
	}

	// Read generated registry
	data, err := os.ReadFile(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var reg object.ModuleRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatal(err)
	}

	// Check pipe.json module is present
	entry, ok := reg.Modules["my-module"]
	if !ok {
		t.Fatal("my-module not found in registry")
	}
	if entry.Latest != "1.2.3" {
		t.Errorf("latest: expected 1.2.3, got %s", entry.Latest)
	}
	if len(entry.Functions) != 2 {
		t.Errorf("functions: expected 2, got %d", len(entry.Functions))
	}
	expectedURL := baseURL + "/my-module/module.pipe"
	if entry.URL != expectedURL {
		t.Errorf("url: expected %s, got %s", expectedURL, entry.URL)
	}
	if entry.Versions["1.2.3"] != expectedURL {
		t.Errorf("versions: expected %s, got %s", expectedURL, entry.Versions["1.2.3"])
	}
	if entry.Description != "A test module" {
		t.Errorf("description: expected 'A test module', got %s", entry.Description)
	}
}

func TestGenerateRegistryPreservesExisting(t *testing.T) {
	dir := t.TempDir()

	// Pre-create a registry.json with a legacy module
	oldReg := object.ModuleRegistry{
		Modules: map[string]object.ModuleEntry{
			"legacy-mod": {
				Description: "Legacy module",
				Functions:   []string{"old_fn"},
				Latest:      "0.5.0",
				Versions: map[string]string{
					"0.5.0": "https://example.com/legacy/module.pipe",
				},
				URL: "https://example.com/legacy/module.pipe",
			},
		},
	}
	oldData, _ := json.MarshalIndent(oldReg, "", "  ")
	os.WriteFile(filepath.Join(dir, "registry.json"), oldData, 0644)

	// Add a new pipe.json module
	modDir := filepath.Join(dir, "new-module")
	os.MkdirAll(modDir, 0755)
	m := Manifest{
		Name:    "new-module",
		Version: "2.0.0",
		Exports: []string{"new_fn"},
	}
	m.Write(modDir)

	if err := GenerateRegistry(dir, "https://example.com"); err != nil {
		t.Fatal(err)
	}

	// Read back
	data, _ := os.ReadFile(filepath.Join(dir, "registry.json"))
	var reg object.ModuleRegistry
	json.Unmarshal(data, &reg)

	// Legacy module preserved
	if _, ok := reg.Modules["legacy-mod"]; !ok {
		t.Error("legacy module should be preserved")
	}
	// New module added
	if _, ok := reg.Modules["new-module"]; !ok {
		t.Error("new module should be added")
	}
}

func TestGenerateRegistryReport(t *testing.T) {
	dir := t.TempDir()

	modDir := filepath.Join(dir, "good-module")
	os.MkdirAll(modDir, 0755)
	m := Manifest{Name: "good-module", Version: "1.0.0", Exports: []string{"a", "b"}}
	m.Write(modDir)

	report, err := GenerateRegistryReport(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range report {
		if strings.Contains(line, "good-module") && strings.Contains(line, "exports") {
			found = true
		}
	}
	if !found {
		t.Error("report should include at least one valid module")
	}
}
