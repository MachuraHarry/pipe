package module

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Manifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Author       string            `json:"author,omitempty"`
	License      string            `json:"license,omitempty"`
	Pipe         string            `json:"pipe,omitempty"`
	Exports      []string          `json:"exports,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

func Parse(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "pipe.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read pipe.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid pipe.json: %w", err)
	}
	return &m, nil
}

func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("pipe.json: 'name' is required")
	}
	if m.Version == "" {
		return fmt.Errorf("pipe.json: 'version' is required")
	}
	for ch := range m.Name {
		if !isNameChar(byte(m.Name[ch])) {
			return fmt.Errorf("pipe.json: 'name' contains invalid character '%c' — use lowercase letters, digits, hyphens, and underscores", m.Name[ch])
		}
	}
	return nil
}

func isNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

func (m *Manifest) Write(dir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join(dir, "pipe.json")
	return os.WriteFile(path, data, 0644)
}

func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "pipe.json"))
	return err == nil
}

func InitModule(dir, name string) error {
	if Exists(dir) {
		return fmt.Errorf("pipe.json already exists in %s", dir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	m := Manifest{
		Name:        name,
		Version:     "0.1.0",
		Description: "TODO: describe your module",
		License:     "MIT",
		Exports:     []string{},
	}
	if err := m.Write(dir); err != nil {
		return err
	}

	modContent := fmt.Sprintf(`--- %s: TODO description
--- Version: 0.1.0

export fn hello name
    "Hello, " ++ name ++ "!"
`, name)
	if err := os.WriteFile(filepath.Join(dir, "module.pipe"), []byte(modContent), 0644); err != nil {
		return err
	}

	readmeContent := fmt.Sprintf(`# %s

TODO: describe what this module does.

## Install

    pipe -install

## Usage

    import "%s"

    print (hello "World")
`, name, name)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeContent), 0644); err != nil {
		return err
	}

	return nil
}
