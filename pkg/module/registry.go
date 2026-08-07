package module

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MachuraHarry/pipe/pkg/object"
)

func GenerateRegistry(dir string, baseURL string) error {
	if baseURL == "" {
		baseURL = "https://raw.githubusercontent.com/MachuraHarry/pipe-modules/master"
	}

	// Load existing registry to preserve legacy modules without pipe.json
	reg := &object.ModuleRegistry{Modules: make(map[string]object.ModuleEntry)}

	existingPath := filepath.Join(dir, "registry.json")
	if data, err := os.ReadFile(existingPath); err == nil {
		var oldReg object.ModuleRegistry
		if err := json.Unmarshal(data, &oldReg); err == nil {
			reg = &oldReg
		}
	}

	// Scan subdirectories for pipe.json
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		modDir := filepath.Join(dir, entry.Name())
		if !Exists(modDir) {
			continue
		}

		m, err := Parse(modDir)
		if err != nil {
			continue
		}
		if err := m.Validate(); err != nil {
			continue
		}

		moduleURL := baseURL + "/" + entry.Name() + "/module.pipe"

		entry := object.ModuleEntry{
			Description: m.Description,
			Functions:   m.Exports,
			Latest:      m.Version,
			Versions: map[string]string{
				m.Version: moduleURL,
			},
			URL: moduleURL,
		}

		// Preserve existing versions
		if existing, ok := reg.Modules[m.Name]; ok {
			if existing.Versions != nil {
				for v, url := range existing.Versions {
					if _, has := entry.Versions[v]; !has {
						entry.Versions[v] = url
					}
				}
			}
		}

		reg.Modules[m.Name] = entry
	}

	// Write registry.json
	out, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	return os.WriteFile(filepath.Join(dir, "registry.json"), out, 0644)
}

func GenerateRegistryReport(dir string) ([]string, error) {
	var report []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	modulesWithJSON := 0
	modulesWithoutJSON := 0

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		modDir := filepath.Join(dir, entry.Name())

		if Exists(modDir) {
			m, err := Parse(modDir)
			if err == nil && m.Validate() == nil {
				modulesWithJSON++
				report = append(report, fmt.Sprintf("  ✓ %s v%s — %d exports", m.Name, m.Version, len(m.Exports)))
			} else {
				modulesWithJSON++
				report = append(report, fmt.Sprintf("  ✗ %s — %v", entry.Name(), err))
			}
		} else {
			modulesWithoutJSON++
			report = append(report, fmt.Sprintf("  ○ %s (no pipe.json, preserved from existing)", entry.Name()))
		}
	}

	sort.Strings(report)
	header := fmt.Sprintf("Scanned %d modules (%d with pipe.json, %d legacy):",
		modulesWithJSON+modulesWithoutJSON, modulesWithJSON, modulesWithoutJSON)
	report = append([]string{header}, report...)

	return report, nil
}
