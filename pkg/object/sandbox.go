package object

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
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
	mu         sync.Mutex
	access     FSAccess
	workingDir string
	tempDir    string
	allowRead  []string
	allowWrite []string
}

// Canonicalize resolves the absolute, symlink-cleaned path of path. For
// temp-only profiles, paths outside the sandbox temp dir are redirected into
// it. The returned path is safe to pass to the filesystem.
func (p *defaultPathPolicy) Canonicalize(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved := resolveSymlinks(abs)

	switch p.access {
	case FSNone:
		return resolved, nil
	case FSTempOnly:
		td := p.tempDir
		if td == "" {
			td = p.ensureTempDirLocked()
		}
		if td == "" {
			return "", fmt.Errorf("E_SANDBOX: temp dir not configured for temp-only access")
		}
		if inside(td, resolved) {
			return resolved, nil
		}
		rel, relErr := filepath.Rel(p.workingDir, resolved)
		if relErr != nil || !inside(td, filepath.Join(td, rel)) {
			rel = filepath.Base(resolved)
		}
		return filepath.Join(td, rel), nil
	default:
		return resolved, nil
	}
}

func (p *defaultPathPolicy) ensureTempDirLocked() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tempDir == "" {
		p.tempDir, _ = filepath.Abs(filepath.Join(p.workingDir, ".pipe_sandbox"))
		if err := os.MkdirAll(p.tempDir, 0755); err != nil {
			if _, sErr := os.Stat(p.tempDir); os.IsNotExist(sErr) {
				p.tempDir = ""
			}
		}
	}
	return p.tempDir
}

func (p *defaultPathPolicy) AllowRead(path string) bool {
	switch p.access {
	case FSFull, FSReadOnly:
		return true
	case FSTempOnly:
		td := p.tempDir
		if td == "" {
			td = p.ensureTempDirLocked()
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		return inside(td, abs)
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
		td := p.tempDir
		if td == "" {
			td = p.ensureTempDirLocked()
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		return inside(td, abs)
	default:
		return false
	}
}

// resolveSymlinks resolves as much of path as exists, cleaning symlinks, and
// re-appends any trailing components that do not exist yet.
func resolveSymlinks(path string) string {
	if r, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(r)
	}
	var missing []string
	cur := path
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				r = filepath.Join(r, missing[i])
			}
			return filepath.Clean(r)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return filepath.Clean(path)
		}
		missing = append(missing, filepath.Base(cur))
		cur = parent
	}
}

// inside reports whether path is equal to root or strictly below it, so that
// "root/../escape" and sibling-directory prefix tricks do not pass.
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

type SandboxProfile struct {
	Name             string
	FSAccess         FSAccess
	Network          bool
	NetworkWhitelist []string
	Exec             bool
	AI               bool
	Timeout          int
	MaxToolCalls     int
	Budget           float64
	AuditLog         bool
	Env              map[string]string
	WorkDir          string
	PathPolicy       PathPolicy

	mu         sync.Mutex
	toolCalls  int
	auditTrail []AuditEntry
	spentCost  float64
	Locked     bool
}

// IsLocked reports whether the profile refuses to be switched away from.
func (p *SandboxProfile) IsLocked() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Locked
}

// SetLocked marks the profile as immutable for the rest of the run.
func (p *SandboxProfile) SetLocked() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Locked = true
}

func NewSandboxProfile(name string) *SandboxProfile {
	return &SandboxProfile{
		Name:     name,
		FSAccess: FSFull,
		Network:  true,
		Exec:     true,
		AI:       true,
		Budget:   0,
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

func (p *SandboxProfile) Canonicalize(path string) (string, error) {
	if p.PathPolicy == nil {
		return path, nil
	}
	return p.PathPolicy.Canonicalize(path)
}

// canonicalRead resolves path and enforces the read policy, returning the
// safe path to pass to the filesystem.
func (p *SandboxProfile) canonicalRead(path string) (string, error) {
	canon, cerr := p.Canonicalize(path)
	if cerr != nil {
		return "", cerr
	}
	if canErr := p.CanRead(canon); canErr != nil {
		return "", canErr
	}
	return canon, nil
}

// canonicalWrite resolves path and enforces the write policy, returning the
// safe path to pass to the filesystem.
func (p *SandboxProfile) canonicalWrite(path string) (string, error) {
	canon, cerr := p.Canonicalize(path)
	if cerr != nil {
		return "", cerr
	}
	if canErr := p.CanWrite(canon); canErr != nil {
		return "", canErr
	}
	return canon, nil
}

func (p *SandboxProfile) CanNetwork() error {
	if !p.Network {
		return fmt.Errorf("E_SANDBOX: network access blocked by profile '%s'", p.Name)
	}
	return nil
}

// IsSubsetOf reports whether sub is no more permissive than super across every
// sandbox dimension. The sandbox can only "ratchet down": once a restricted
// profile is active, switching to (or registering) a profile that grants more
// rights is rejected. A profile with an empty network whitelist (allow all) is
// treated as the least restrictive case and therefore a superset of any
// whitelist.
func (sub *SandboxProfile) IsSubsetOf(super *SandboxProfile) bool {
	if sub.FSAccess > super.FSAccess {
		return false
	}
	if sub.Network && !super.Network {
		return false
	}
	if sub.Exec && !super.Exec {
		return false
	}
	if sub.AI && !super.AI {
		return false
	}
	if len(super.NetworkWhitelist) > 0 {
		for _, entry := range sub.NetworkWhitelist {
			host, port, path := splitNetworkTarget(entry)
			if host == "" {
				return false
			}
			matched := false
			for _, pattern := range super.NetworkWhitelist {
				if matchNetworkPattern(pattern, host, port, path) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	return true
}

// ratchetError builds the error returned when a profile change would escalate
// beyond the rights of the currently active profile.
func ratchetError(target, current string) error {
	return fmt.Errorf("E_SANDBOX: profile '%s' is not a subset of active profile '%s'; the sandbox can only ratchet down", target, current)
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
	if p.Budget > 0 {
		return p.checkBudget(ai.EstimateMaxCost(4096))
	}
	return nil
}

// checkBudget blocks a call whose estimated cost would exceed the remaining
// budget. The estimate is a conservative upper bound, so this prevents a
// single call from blowing through the budget before its cost is recorded.
func (p *SandboxProfile) checkBudget(estimated float64) error {
	p.mu.Lock()
	spent := p.spentCost
	p.mu.Unlock()
	if spent+estimated > p.Budget {
		return fmt.Errorf("E_SANDBOX: budget exceeded (%.4f USD) in profile '%s'", p.Budget, p.Name)
	}
	return nil
}

func (p *SandboxProfile) RecordCost(costUSD float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spentCost += costUSD
}

func (p *SandboxProfile) CanToolCall() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.MaxToolCalls > 0 && p.toolCalls >= p.MaxToolCalls {
		return fmt.Errorf("E_SANDBOX: max tool calls (%d) exceeded in profile '%s'", p.MaxToolCalls, p.Name)
	}
	p.toolCalls++
	return nil
}

func (p *SandboxProfile) CanNetworkTo(target string) error {
	if !p.Network {
		return fmt.Errorf("E_SANDBOX: network access blocked by profile '%s'", p.Name)
	}
	if len(p.NetworkWhitelist) == 0 {
		return nil
	}
	host, port, path := splitNetworkTarget(target)
	if host == "" {
		return fmt.Errorf("E_SANDBOX: network target '%s' not in whitelist of profile '%s'", target, p.Name)
	}
	for _, pattern := range p.NetworkWhitelist {
		if matchNetworkPattern(pattern, host, port, path) {
			return nil
		}
	}
	return fmt.Errorf("E_SANDBOX: network target '%s' not in whitelist of profile '%s'", target, p.Name)
}

// splitNetworkTarget parses either an absolute URL ("https://host:443/path")
// or a bare "host[:port][/path]" into its components.
func splitNetworkTarget(target string) (host, port, path string) {
	raw := target
	if !strings.Contains(raw, "://") {
		raw = "//" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", ""
	}
	host = u.Hostname()
	if host == "" {
		return "", "", ""
	}
	port = u.Port()
	path = u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if port == "" {
		switch strings.ToLower(u.Scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return host, port, path
}

// matchNetworkPattern checks one whitelist entry. An entry may be a bare
// hostname (matches the host and its subdomains), a host:port pair, or a
// host/path prefix.
func matchNetworkPattern(pattern, host, port, path string) bool {
	ph, pport, ppath := splitNetworkTarget(pattern)
	if ph == "" {
		return host == pattern
	}
	if !hostMatches(ph, host) {
		return false
	}
	if pport != "" && port != "" && pport != port {
		return false
	}
	if ppath != "" && ppath != "/" {
		if !strings.HasPrefix(path, ppath) {
			return false
		}
	}
	return true
}

// hostMatches reports whether host equals pattern or is a subdomain of it.
func hostMatches(pattern, host string) bool {
	if pattern == host {
		return true
	}
	if strings.HasSuffix(host, "."+pattern) {
		return true
	}
	return false
}

func (p *SandboxProfile) Audit(event, detail string) {
	if !p.AuditLog {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.auditTrail = append(p.auditTrail, AuditEntry{
		Time:    time.Now(),
		Event:   event,
		Detail:  detail,
		Profile: p.Name,
	})
}

func (p *SandboxProfile) GetAuditLog() []AuditEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make([]AuditEntry, len(p.auditTrail))
	copy(cp, p.auditTrail)
	return cp
}

func (p *SandboxProfile) GetSpentBudget() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.spentCost
}

type AuditEntry struct {
	Time    time.Time
	Event   string
	Detail  string
	Profile string
}

func (p *SandboxProfile) ensureTempDir() string {
	if p.FSAccess != FSTempOnly {
		return ""
	}
	if dpp, ok := p.PathPolicy.(*defaultPathPolicy); ok {
		return dpp.ensureTempDirLocked()
	}
	return ""
}

// ActiveProfile is the sandbox profile currently in force. It is read from
// every sandboxed builtin, so it is swapped atomically.
var ActiveProfile atomic.Pointer[SandboxProfile]

var (
	profileRegistryMu sync.RWMutex
	profileRegistry   = map[string]*SandboxProfile{
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
			PathPolicy: &defaultPathPolicy{access: FSTempOnly, workingDir: "."},
		},
	}
)

func init() {
	ActiveProfile.Store(profileRegistry["none"])
}

func RegisterProfile(name string, p *SandboxProfile) error {
	profileRegistryMu.Lock()
	defer profileRegistryMu.Unlock()
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
	profileRegistryMu.RLock()
	defer profileRegistryMu.RUnlock()
	p, ok := profileRegistry[name]
	if !ok {
		return nil, fmt.Errorf("E_SANDBOX: unknown profile '%s'", name)
	}
	return p, nil
}

func WithProfile(name string, fn func() Object) Object {
	prev := ActiveProfile.Load()
	defer func() { ActiveProfile.Store(prev) }()

	prof, profErr := GetProfile(name)
	if profErr != nil {
		return &Error{Message: profErr.Error()}
	}
	ActiveProfile.Store(prof)
	return fn()
}

// withActiveProfile runs fn with p as the active profile, then restores the
// previous one. Unlike WithProfile it does not require p to be registered.
func withActiveProfile(p *SandboxProfile, fn func() Object) Object {
	prev := ActiveProfile.Load()
	ActiveProfile.Store(p)
	defer func() { ActiveProfile.Store(prev) }()
	return fn()
}

func sandboxError(feature string) *Error {
	msg := "E_SANDBOX: " + feature + " is blocked by active profile '" + ActiveProfile.Load().Name + "'"
	return &Error{Message: msg}
}
