package object

import (
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

func ResolveImport(path string) (string, string, error) {
	return ResolveImportFrom(path, "")
}

func ResolveImportFrom(path string, sourceFile string) (string, string, error) {
	// URL import
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return fetchURLModule(path)
	}

	// Build search dirs: importing file's directory first, then CWD, then PIPE_PATH
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
