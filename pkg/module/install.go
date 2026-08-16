package module

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/object"
)

type LockEntry struct {
	Version      string            `json:"version"`
	URL          string            `json:"url,omitempty"`
	Checksum     string            `json:"checksum,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

type Lockfile struct {
	Modules map[string]LockEntry `json:"modules"`
}

func Install(dir string) error {
	m, err := Parse(dir)
	if err != nil {
		return fmt.Errorf("cannot read pipe.json: %w", err)
	}

	if len(m.Dependencies) == 0 {
		fmt.Println("No dependencies to install.")
		return nil
	}

	lock := &Lockfile{Modules: make(map[string]LockEntry)}
	existing, _ := ReadLockfile(dir)

	fmt.Printf("Installing dependencies for %s…\n", m.Name)
	if err := resolveDeps(m.Dependencies, lock, nil, existing); err != nil {
		return err
	}

	for name, entry := range lock.Modules {
		modDir := filepath.Join(object.ModuleCacheDir(), name)
		os.MkdirAll(modDir, 0755)

		content, err := fetchModule(entry.URL)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", name, err)
		}

		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
		if entry.Checksum != "" && entry.Checksum != checksum {
			return fmt.Errorf("checksum mismatch for %s v%s: lockfile has %s, fetched %s", name, entry.Version, entry.Checksum, checksum)
		}
		entry.Checksum = checksum

		modPath := filepath.Join(modDir, "module.pipe")
		if err := os.WriteFile(modPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", modPath, err)
		}

		lock.Modules[name] = entry

		fmt.Printf("  ✓ %s v%s\n", name, entry.Version)
	}

	fmt.Printf("\nSaved lockfile to pipe.lock\n")
	return WriteLockfile(dir, lock)
}

func resolveDeps(deps map[string]string, lock *Lockfile, path []string, existing *Lockfile) error {
	for depName, depVersion := range deps {
		if _, ok := lock.Modules[depName]; ok {
			continue
		}

		for _, p := range path {
			if p == depName {
				return fmt.Errorf("circular dependency: %s", strings.Join(append(path, depName), " → "))
			}
		}

		// Prefer a pinned entry from an existing lockfile for reproducibility.
		if existing != nil {
			if pinned, ok := existing.Modules[depName]; ok && pinned.URL != "" {
				lock.Modules[depName] = pinned
				if len(pinned.Dependencies) > 0 {
					newPath := append(path, depName)
					if err := resolveDeps(pinned.Dependencies, lock, newPath, existing); err != nil {
						return err
					}
				}
				continue
			}
		}

		reg, err := object.FetchRegistry()
		if err != nil {
			return fmt.Errorf("cannot fetch registry: %w", err)
		}
		mod, ok := reg.Modules[depName]
		if !ok {
			return fmt.Errorf("module not found: %s", depName)
		}

		bestVersion := resolveVersion(depVersion, &mod)
		bestURL := mod.Versions[bestVersion]
		if bestURL == "" {
			bestURL = mod.URL
		}
		if bestURL == "" {
			return fmt.Errorf("no URL found for %s %s", depName, depVersion)
		}

		entry := LockEntry{
			Version: bestVersion,
			URL:     bestURL,
		}

		depManifest := fetchModuleManifest(bestURL)
		if depManifest != nil && len(depManifest.Dependencies) > 0 {
			entry.Dependencies = depManifest.Dependencies
			lock.Modules[depName] = entry
			newPath := append(path, depName)
			if err := resolveDeps(depManifest.Dependencies, lock, newPath, existing); err != nil {
				return err
			}
		}

		lock.Modules[depName] = entry
	}
	return nil
}

func resolveVersion(constraint string, mod *object.ModuleEntry) string {
	c := strings.TrimSpace(constraint)
	if c == "" || c == "*" || c == "latest" {
		return mod.Latest
	}
	if strings.HasPrefix(c, "^") {
		// Caret constraint: >= base with same major version (< major+1).
		base := strings.TrimPrefix(c, "^")
		baseVer := parseSemver(base)
		if baseVer.major < 0 {
			return mod.Latest
		}
		best := ""
		for v := range mod.Versions {
			if !versionLTE(base, v) {
				continue
			}
			vv := parseSemver(v)
			if vv.major != baseVer.major {
				continue
			}
			if best == "" || versionLTE(best, v) {
				best = v
			}
		}
		if best != "" {
			return best
		}
		return mod.Latest
	}
	if _, ok := mod.Versions[c]; ok {
		return c
	}
	return mod.Latest
}

func fetchModuleManifest(moduleURL string) *Manifest {
	pipeJSONURL := strings.Replace(moduleURL, "/module.pipe", "/pipe.json", 1)
	_, content, err := fetchModuleRaw(pipeJSONURL)
	if err != nil {
		return nil
	}
	var m Manifest
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return nil
	}
	return &m
}

var fetchCache = make(map[string]string)

func fetchModule(url string) (string, error) {
	if content, ok := fetchCache[url]; ok {
		return content, nil
	}
	_, content, err := fetchModuleRaw(url)
	if err != nil {
		return "", err
	}
	fetchCache[url] = content
	return content, nil
}

func fetchModuleRaw(url string) (string, string, error) {
	return object.ResolveImport(url)
}

func WriteLockfile(dir string, lock *Lockfile) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "pipe.lock"), data, 0644)
}

func ReadLockfile(dir string) (*Lockfile, error) {
	path := filepath.Join(dir, "pipe.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lock Lockfile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	return &lock, nil
}
