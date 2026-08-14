package module

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/object"
)

const (
	DefaultModulesRepo = "https://github.com/MachuraHarry/pipe-modules.git"
	DefaultModulesRaw  = "https://raw.githubusercontent.com/MachuraHarry/pipe-modules/master"
)

var versionRegexp = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// Publish pushes a module to the shared registry via a pull request.
//
// Workflow:
//  1. Validate the module's pipe.json.
//  2. Check for a duplicate version in the current registry.
//  3. Clone the registry repo, copy the module into modules/<name>.
//  4. Add an entry to registry.json.
//  5. Commit on branch publish/<name>-<version> and open a PR with gh.
func Publish(dir string) error {
	if dir == "" {
		dir = "."
	}
	m, err := Parse(dir)
	if err != nil {
		return fmt.Errorf("cannot read pipe.json: %w", err)
	}
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid pipe.json: %w", err)
	}
	if !versionRegexp.MatchString(m.Version) {
		return fmt.Errorf("invalid version %q — use semantic versioning (e.g. 1.2.0)", m.Version)
	}
	if _, err := os.Stat(filepath.Join(dir, "module.pipe")); err != nil {
		return fmt.Errorf("module.pipe not found in %s — create it with pipe -init", dir)
	}

	if err := checkDuplicateVersion(m.Name, m.Version); err != nil {
		return err
	}
	if err := checkGH(); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "pipe-publish-*")
	if err != nil {
		return fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	if err := cloneRepo(tmp); err != nil {
		return err
	}

	modsDir := filepath.Join(tmp, "repo", "modules")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return err
	}

	target := filepath.Join(modsDir, m.Name)
	if err := copyDir(dir, target); err != nil {
		return err
	}

	if err := updateRegistry(filepath.Join(tmp, "repo", "registry.json"), m); err != nil {
		return err
	}

	branch := fmt.Sprintf("publish/%s-%s", m.Name, m.Version)
	if err := commitAndPR(filepath.Join(tmp, "repo"), branch, m); err != nil {
		return err
	}

	return nil
}

func checkDuplicateVersion(name, version string) error {
	reg, err := object.FetchRegistry()
	if err != nil {
		return fmt.Errorf("cannot fetch registry: %w", err)
	}
	if mod, ok := reg.Modules[name]; ok {
		if _, has := mod.Versions[version]; has {
			return fmt.Errorf("version %s of %s already exists in the registry — bump the version in pipe.json", version, name)
		}
	}
	return nil
}

func checkGH() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found — install it from https://cli.github.com and run 'gh auth login'")
	}
	cmd := exec.Command("gh", "auth", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh is not authenticated: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func cloneRepo(tmp string) error {
	fmt.Println("Cloning module registry…")
	cmd := exec.Command("git", "clone", "--depth", "1", DefaultModulesRepo, filepath.Join(tmp, "repo"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot clone registry: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		// Never copy the lockfile or the module's own registry artifacts.
		if e.Name() == "pipe.lock" {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateRegistry(regPath string, m *Manifest) error {
	reg := &object.ModuleRegistry{Modules: make(map[string]object.ModuleEntry)}

	if data, err := os.ReadFile(regPath); err == nil {
		if err := json.Unmarshal(data, reg); err != nil {
			return fmt.Errorf("invalid registry.json: %w", err)
		}
	}

	moduleURL := DefaultModulesRaw + "/modules/" + m.Name + "/module.pipe"
	entry := object.ModuleEntry{
		Description: m.Description,
		Functions:   m.Exports,
		Latest:      m.Version,
		Versions: map[string]string{
			m.Version: moduleURL,
		},
		URL: moduleURL,
	}

	// Preserve existing versions and merge
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

	out, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(regPath, out, 0644)
}

func commitAndPR(repoDir, branch string, m *Manifest) error {
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
		}
		return string(out), nil
	}

	if _, err := run("checkout", "-b", branch); err != nil {
		return fmt.Errorf("cannot create branch %s: %s", branch, err)
	}
	if _, err := run("add", "-A"); err != nil {
		return fmt.Errorf("cannot stage files: %s", err)
	}
	// Ensure a commit identity for this repository only; the commit will be
	// attributed via gh when the PR is merged.
	_, _ = run("config", "user.name", "pipe-publish")
	_, _ = run("config", "user.email", "pipe-publish@users.noreply.github.com")
	msg := fmt.Sprintf("Add %s v%s", m.Name, m.Version)
	if _, err := run("commit", "-m", msg); err != nil {
		return fmt.Errorf("cannot commit: %s", err)
	}
	if _, err := run("push", "--set-upstream", "origin", branch); err != nil {
		return fmt.Errorf("cannot push branch %s: %s", branch, err)
	}

	prBody := fmt.Sprintf(
		"Adds `%s` v%s to the Pipe module registry.\n\n- Description: %s\n- License: %s\n- Exports: %s",
		m.Name, m.Version, m.Description, m.License, strings.Join(m.Exports, ", "),
	)
	cmd := exec.Command("gh", "pr", "create",
		"--repo", "MachuraHarry/pipe-modules",
		"--head", branch,
		"--title", fmt.Sprintf("publish: %s v%s", m.Name, m.Version),
		"--body", prBody)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot open pull request: %s", strings.TrimSpace(string(out)))
	}
	fmt.Printf("Pull request created: %s\n", strings.TrimSpace(string(out)))
	return nil
}
