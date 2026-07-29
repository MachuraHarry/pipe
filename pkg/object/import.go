package object

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var moduleCacheDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	moduleCacheDir = filepath.Join(home, ".pipe", "modules")
	os.MkdirAll(moduleCacheDir, 0755)
}

func ModuleCacheDir() string { return moduleCacheDir }

const RegistryURL = "https://raw.githubusercontent.com/MachuraHarry/pipe-modules/master/registry.json"

type ModuleEntry struct {
	Description string            `json:"description"`
	Functions   []string          `json:"functions"`
	URL         string            `json:"url"`
	Latest      string            `json:"latest"`
	Versions    map[string]string `json:"versions"`
}

type ModuleRegistry struct {
	Modules map[string]ModuleEntry `json:"modules"`
}

func FetchRegistry() (*ModuleRegistry, error) {
	url := RegistryURL + "?t=" + fmt.Sprintf("%d", time.Now().Unix())
	resp, err := httpGet(url)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch registry: %w", err)
	}
	var reg ModuleRegistry
	if err := json.Unmarshal([]byte(resp), &reg); err != nil {
		return nil, fmt.Errorf("invalid registry: %w", err)
	}
	return &reg, nil
}

func resolveRegistryModule(name, version string) (string, error) {
	reg, err := FetchRegistry()
	if err != nil {
		return "", err
	}
	mod, ok := reg.Modules[name]
	if !ok {
		return "", fmt.Errorf("module not found: %s", name)
	}

	if version != "" {
		if mod.Versions != nil {
			if url, ok := mod.Versions[version]; ok {
				return url, nil
			}
		}
		return "", fmt.Errorf("version %s not found for module %s", version, name)
	}

	if mod.Versions != nil && mod.Latest != "" {
		if url, ok := mod.Versions[mod.Latest]; ok {
			return url, nil
		}
	}

	if mod.URL != "" {
		return mod.URL, nil
	}

	return "", fmt.Errorf("no URL found for module %s", name)
}

func parseModuleSpec(path string) (name, version string) {
	if idx := strings.LastIndex(path, "@"); idx > 0 {
		return path[:idx], path[idx+1:]
	}
	return path, ""
}

func ParseModuleSpec(path string) (name, version string) {
	return parseModuleSpec(path)
}

func ResolveModuleURL(name, version string) (string, error) {
	return resolveRegistryModule(name, version)
}

func ResolveImport(path string) (string, string, error) {
	return ResolveImportFrom(path, "")
}

func ResolveImportFrom(path string, sourceFile string) (string, string, error) {
	// URL import
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return fetchURLModule(path)
	}

	name, version := parseModuleSpec(path)

	// Try local file first (exact path, then in search dirs)
	searchDirs := []string{}
	if sourceFile != "" {
		searchDirs = append(searchDirs, filepath.Dir(sourceFile))
	}
	searchDirs = append(searchDirs, ".")
	searchDirs = append(searchDirs, moduleCacheDir)

	pipePath := os.Getenv("PIPE_PATH")
	if pipePath != "" {
		searchDirs = append(searchDirs, strings.Split(pipePath, ":")...)
	}

	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, path)
		if data, err := os.ReadFile(candidate); err == nil {
			return path, string(data), nil
		}
	}

	// If not found locally and it looks like a module name, try registry
	if !strings.Contains(name, "/") && !strings.Contains(name, ".") && !strings.HasSuffix(name, ".pipe") {
		if url, err := resolveRegistryModule(name, version); err == nil {
			return fetchURLModule(url)
		}
	}

	return "", "", fmt.Errorf("import not found: %s (searched: %v)", path, searchDirs)
}

func fetchURLModule(url string) (string, string, error) {
	// Check cache first
	cacheKey := urlToCacheKey(url)
	cachePath := filepath.Join(moduleCacheDir, cacheKey)
	if data, err := os.ReadFile(cachePath); err == nil {
		return url, string(data), nil
	}

	// Fetch from URL
	resp, err := httpGet(url)
	if err != nil {
		return "", "", fmt.Errorf("fetch %s: %w", url, err)
	}

	// Cache the result
	os.WriteFile(cachePath, []byte(resp), 0644)

	return url, resp, nil
}

func urlToCacheKey(url string) string {
	// Convert URL to a safe filename: replace special chars
	url = strings.ReplaceAll(url, "https://", "")
	url = strings.ReplaceAll(url, "http://", "")
	url = strings.ReplaceAll(url, "/", "_")
	url = strings.ReplaceAll(url, ".", "_")
	if len(url) > 100 {
		url = url[:100]
	}
	return url + ".pipe"
}

func httpGet(url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return string(body), nil
}
