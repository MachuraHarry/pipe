package object

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FSAccess int

const (
	FSNone FSAccess = iota
	FSReadOnly
	FSTempOnly
	FSFull
)

func (f FSAccess) String() string {
	switch f {
	case FSNone:
		return "none"
	case FSReadOnly:
		return "read-only"
	case FSTempOnly:
		return "temp-only"
	case FSFull:
		return "full"
	default:
		return "unknown"
	}
}

func ParseFSAccess(s string) (FSAccess, error) {
	switch s {
	case "none":
		return FSNone, nil
	case "read-only":
		return FSReadOnly, nil
	case "temp-only":
		return FSTempOnly, nil
	case "full":
		return FSFull, nil
	default:
		return FSNone, fmt.Errorf("E_SANDBOX: unknown fs access level '%s'", s)
	}
}

type PathPolicy interface {
	Canonicalize(path string) (string, error)
	AllowRead(path string) bool
	AllowWrite(path string) bool
}

type defaultPathPolicy struct {
	access     FSAccess
	workingDir string
	tempDir    string
	allowRead  []string
	allowWrite []string
}

func (p *defaultPathPolicy) Canonicalize(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	switch p.access {
	case FSNone:
		return "", fmt.Errorf("E_SANDBOX: filesystem access denied")
	case FSTempOnly:
		if p.tempDir != "" && !strings.HasPrefix(abs, p.tempDir) {
			rel, err := filepath.Rel(p.workingDir, abs)
			if err != nil {
				return filepath.Join(p.tempDir, filepath.Base(abs)), nil
			}
			return filepath.Join(p.tempDir, rel), nil
		}
	}

	return abs, nil
}

func (p *defaultPathPolicy) AllowRead(path string) bool {
	switch p.access {
	case FSFull, FSReadOnly:
		return true
	case FSTempOnly:
		if p.tempDir == "" {
			return false
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		return strings.HasPrefix(abs, p.tempDir)
	case FSNone:
		return false
	}
	return false
}

func (p *defaultPathPolicy) AllowWrite(path string) bool {
	switch p.access {
	case FSFull:
		for _, pattern := range p.allowWrite {
			if matched, _ := filepath.Match(pattern, path); matched {
				return true
			}
		}
		return len(p.allowWrite) == 0
	case FSTempOnly:
		if p.tempDir == "" {
			return false
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		return strings.HasPrefix(abs, p.tempDir)
	default:
		return false
	}
}

type SandboxProfile struct {
	Name       string
	FSAccess   FSAccess
	Network    bool
	Exec       bool
	AI         bool
	Timeout    int
	Env        map[string]string
	WorkDir    string
	PathPolicy PathPolicy
}

func NewSandboxProfile(name string) *SandboxProfile {
	return &SandboxProfile{
		Name:     name,
		FSAccess: FSFull,
		Network:  true,
		Exec:     true,
		AI:       true,
		Env:      make(map[string]string),
		WorkDir:  ".",
	}
}

func (p *SandboxProfile) CanRead(path string) error {
	if !p.PathPolicy.AllowRead(path) {
		return fmt.Errorf("E_SANDBOX: read of '%s' blocked by profile '%s' (fs: %s)", path, p.Name, p.FSAccess)
	}
	return nil
}

func (p *SandboxProfile) CanWrite(path string) error {
	if !p.PathPolicy.AllowWrite(path) {
		return fmt.Errorf("E_SANDBOX: write of '%s' blocked by profile '%s' (fs: %s)", path, p.Name, p.FSAccess)
	}
	return nil
}

func (p *SandboxProfile) CanNetwork() error {
	if !p.Network {
		return fmt.Errorf("E_SANDBOX: network access blocked by profile '%s'", p.Name)
	}
	return nil
}

func (p *SandboxProfile) CanExec() error {
	if !p.Exec {
		return fmt.Errorf("E_SANDBOX: exec blocked by profile '%s'", p.Name)
	}
	return nil
}

func (p *SandboxProfile) CanAI() error {
	if !p.AI {
		return fmt.Errorf("E_SANDBOX: AI calls blocked by profile '%s'", p.Name)
	}
	return nil
}

func (p *SandboxProfile) ensureTempDir() string {
	if p.FSAccess != FSTempOnly {
		return ""
	}
	pp := p.PathPolicy
	if dpp, ok := pp.(*defaultPathPolicy); ok {
		if dpp.tempDir == "" {
			dpp.tempDir = filepath.Join(p.WorkDir, ".pipe_sandbox")
			os.MkdirAll(dpp.tempDir, 0755)
		}
		return dpp.tempDir
	}
	return ""
}

var (
	profileRegistry = map[string]*SandboxProfile{
		"none": {
			Name: "none", FSAccess: FSFull, Network: true, Exec: true, AI: true,
			Env: make(map[string]string), WorkDir: ".",
			PathPolicy: &defaultPathPolicy{access: FSFull},
		},
		"strict": {
			Name: "strict", FSAccess: FSReadOnly, Network: false, Exec: false, AI: false,
			Env: make(map[string]string), WorkDir: ".",
			PathPolicy: &defaultPathPolicy{access: FSReadOnly},
		},
		"noexec": {
			Name: "noexec", FSAccess: FSFull, Network: false, Exec: false, AI: false,
			Env: make(map[string]string), WorkDir: ".",
			PathPolicy: &defaultPathPolicy{access: FSFull},
		},
		"isolated": {
			Name: "isolated", FSAccess: FSNone, Network: false, Exec: false, AI: false,
			Env: make(map[string]string), WorkDir: ".",
			PathPolicy: &defaultPathPolicy{access: FSNone},
		},
		"networked": {
			Name: "networked", FSAccess: FSTempOnly, Network: true, Exec: false, AI: true,
			Env: make(map[string]string), WorkDir: ".",
			PathPolicy: &defaultPathPolicy{access: FSTempOnly},
		},
	}
	ActiveProfile = profileRegistry["none"]
)

func RegisterProfile(name string, p *SandboxProfile) error {
	if _, exists := profileRegistry[name]; exists {
		return fmt.Errorf("E_SANDBOX: profile '%s' already exists", name)
	}
	if p.PathPolicy == nil {
		p.PathPolicy = &defaultPathPolicy{
			access:     p.FSAccess,
			workingDir: p.WorkDir,
			allowRead:  nil,
			allowWrite: nil,
		}
	}
	if p.Env == nil {
		p.Env = make(map[string]string)
	}
	profileRegistry[name] = p
	return nil
}

func GetProfile(name string) (*SandboxProfile, error) {
	p, ok := profileRegistry[name]
	if !ok {
		return nil, fmt.Errorf("E_SANDBOX: unknown profile '%s'", name)
	}
	return p, nil
}

func WithProfile(name string, fn func() Object) Object {
	prev := ActiveProfile
	defer func() { ActiveProfile = prev }()

	prof, profErr := GetProfile(name)
	if profErr != nil {
		return &Error{Message: profErr.Error()}
	}
	ActiveProfile = prof
	return fn()
}

func sandboxError(feature string) *Error {
	msg := "E_SANDBOX: " + feature + " is blocked by active profile '" + ActiveProfile.Name + "'"
	return &Error{Message: msg}
}
