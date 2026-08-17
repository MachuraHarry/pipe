package object

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/parser"
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

	// Absolute filesystem path: read it directly, honoring the sandbox read
	// policy so import cannot bypass read_file restrictions.
	if filepath.IsAbs(path) {
		p := ActiveProfile.Load()
		if p != nil && p.Name != "none" {
			canon, cerr := p.canonicalRead(path)
			if cerr != nil {
				return "", "", cerr
			}
			path = canon
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", "", fmt.Errorf("import not found: %s (searched: absolute path)", path)
		}
		return path, string(data), nil
	}

	name, version := parseModuleSpec(path)

	// Relative imports (./ or ../): resolve against the importing file's
	// directory (or the working directory when there is no source file).
	// This is a dedicated path so relative resolution is unambiguous and
	// never falls through to registry lookup.
	if strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../") {
		base := "."
		if sourceFile != "" {
			base = filepath.Dir(sourceFile)
		}
		candidate := filepath.Join(base, name)
		if res, content, err := readImportCandidate(candidate); err == nil {
			return res, content, nil
		}
		return "", "", fmt.Errorf("import not found: %s (relative to %s)", name, base)
	}

	// Local file or directory (init.pipe) in search dirs
	searchDirs := []string{}
	if sourceFile != "" {
		searchDirs = append(searchDirs, filepath.Dir(sourceFile))
	}
	searchDirs = append(searchDirs, ".")
	searchDirs = append(searchDirs, moduleCacheDir)

	pipePath := os.Getenv("PIPE_PATH")
	if pipePath != "" {
		searchDirs = append(searchDirs, filepath.SplitList(pipePath)...)
	}

	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, path)
		if res, content, err := readImportCandidate(candidate); err == nil {
			return res, content, nil
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

// readImportCandidate reads an import candidate. If the candidate is a
// directory, the module entry point init.pipe is loaded instead, so
// `import "mylib/"` and `import "mylib"` (when mylib is a directory) both
// resolve to mylib/init.pipe. The returned path always points at the file
// actually loaded (init.pipe for directory imports), which keeps cache keys
// and deduplication unambiguous.
func readImportCandidate(candidate string) (string, string, error) {
	// Normalize trailing slashes so path handling is consistent.
	clean := filepath.Clean(candidate)
	if info, err := os.Stat(clean); err == nil && info.IsDir() {
		// Try init.pipe first (convention for user-defined modules),
		// then module.pipe (convention used by pipe -install / registry).
		for _, entry := range []string{"init.pipe", "module.pipe"} {
			entryPath := filepath.Join(clean, entry)
			data, err := os.ReadFile(entryPath)
			if err == nil {
				return entryPath, string(data), nil
			}
		}
		return "", "", fmt.Errorf("directory %s has no init.pipe or module.pipe", clean)
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		return "", "", err
	}
	return clean, string(data), nil
}

func fetchURLModule(url string) (string, string, error) {
	// Check cache first
	cacheKey := urlToCacheKey(url)
	cachePath := filepath.Join(moduleCacheDir, cacheKey)
	if data, err := os.ReadFile(cachePath); err == nil {
		return url, string(data), nil
	}

	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanNetworkTo(url); canErr != nil {
			return "", "", canErr
		}
	} else if Sandbox.Enabled && !Sandbox.AllowNet {
		return "", "", fmt.Errorf("E_SANDBOX: network access blocked for module import")
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
	client := sandboxHTTPClient(15 * time.Second)
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

func ResolveAndParse(path string, sourceFile string) (resolvedPath string, program *ast.Program, err error) {
	resolvedPath, content, err := ResolveImportFrom(path, sourceFile)
	if err != nil {
		return "", nil, err
	}
	program, err = ParseContent(content)
	if err != nil {
		return resolvedPath, nil, fmt.Errorf("parse errors in %s: %v", resolvedPath, err)
	}
	return resolvedPath, program, nil
}

func ParseContent(content string) (*ast.Program, error) {
	l := lexer.New(content)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("%v", p.Errors())
	}
	return program, nil
}
