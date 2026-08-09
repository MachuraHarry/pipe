package object

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempOnlyProfile(t *testing.T) (*SandboxProfile, string) {
	t.Helper()
	dir := t.TempDir()
	p := &SandboxProfile{
		Name:       "tmp-test",
		FSAccess:   FSTempOnly,
		Network:    false,
		Exec:       false,
		AI:         false,
		Env:        map[string]string{},
		WorkDir:    dir,
		PathPolicy: &defaultPathPolicy{access: FSTempOnly, workingDir: dir},
	}
	return p, dir
}

func withProfile(p *SandboxProfile) func() {
	prev := ActiveProfile.Load()
	ActiveProfile.Store(p)
	return func() { ActiveProfile.Store(prev) }
}

// ---- temp-only redirects ----

func TestTempOnlyRedirect(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	canon, err := p.Canonicalize(filepath.Join(dir, "outside.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, ".pipe_sandbox", "outside.txt")
	if canon != want {
		t.Fatalf("expected redirect to %q, got %q", want, canon)
	}
}

func TestTempOnlyWriteRedirectsNotOutside(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	outside := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(outside, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	res := bWriteFile(&String{Value: outside}, &String{Value: "sneaky"})
	if res.Type() == ERROR {
		t.Fatalf("write_file should redirect under temp-only, got error: %s", res.Inspect())
	}

	data, _ := os.ReadFile(outside)
	if string(data) != "original" {
		t.Fatalf("outside file was modified: %q", data)
	}

	sandboxed := filepath.Join(dir, ".pipe_sandbox", "secret.txt")
	got, err := os.ReadFile(sandboxed)
	if err != nil || string(got) != "sneaky" {
		t.Fatalf("expected sandboxed copy to be rewritten, got %q (err=%v)", got, err)
	}
}

func TestTempOnlyParentEscape(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	canon, err := p.Canonicalize(filepath.Join(dir, "..", "evil.txt"))
	if err != nil {
		t.Fatal(err)
	}
	td := filepath.Join(dir, ".pipe_sandbox")
	if !inside(td, canon) {
		t.Fatalf("canonical path escaped the sandbox temp dir: %q", canon)
	}
	if filepath.Clean(canon) == filepath.Join(filepath.Dir(dir), "evil.txt") {
		t.Fatalf("parent escape resolved to original path: %q", canon)
	}
}

func TestTempOnlyPrefixSiblingNotAllowed(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	// Create a sibling dir with a prefix-like name next to the sandbox dir.
	sibling := dir + "x"
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	canon, err := p.Canonicalize(filepath.Join(sibling, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	td := filepath.Join(dir, ".pipe_sandbox")
	if !inside(td, canon) {
		t.Fatalf("sibling dir was treated as inside sandbox: %q", canon)
	}
}

func TestTempOnlySymlinkEscapeBlocked(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	defer withProfile(p)()

	target := filepath.Join(dir, "outside_secret.txt")
	if err := os.WriteFile(target, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	td := filepath.Join(dir, ".pipe_sandbox")
	if err := os.MkdirAll(td, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(td, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not available")
	}

	canon, err := p.Canonicalize(link)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(canon) == filepath.Clean(target) {
		t.Fatalf("symlink path resolved back to the outside file: %q", canon)
	}

	res := bReadFile(&String{Value: link})
	if s, ok := res.(*String); ok && s.Value == "secret" {
		t.Fatal("symlink escape succeeded: read outside file through the sandbox")
	}
}

// ---- read gating ----

func TestReadLinesBlockedByIsolated(t *testing.T) {
	p := NewSandboxProfile("iso")
	p.FSAccess = FSNone
	p.PathPolicy = &defaultPathPolicy{access: FSNone}
	defer withProfile(p)()

	res := bReadLines(&String{Value: "/etc/hostname"})
	if res.Type() != ERROR {
		t.Fatalf("read_lines must be blocked by an isolated profile, got: %s", res.Inspect())
	}
}

func TestEnvSecretBlocked(t *testing.T) {
	res := bEnv(&String{Value: "OPENAI_API_KEY"})
	if res.Type() != ERROR {
		t.Fatalf("env of a secret variable must be blocked, got: %s", res.Inspect())
	}
}

func TestEnvProfileOnly(t *testing.T) {
	p := NewSandboxProfile("envtest")
	p.FSAccess = FSFull
	p.PathPolicy = &defaultPathPolicy{access: FSFull}
	p.Env = map[string]string{"FOO": "bar"}
	defer withProfile(p)()

	res := bEnv(&String{Value: "FOO"})
	if s, ok := res.(*String); !ok || s.Value != "bar" {
		t.Fatalf("env FOO = %s, want profile value", res.Inspect())
	}
	if res := bEnv(&String{Value: "PATH"}); res.Type() != NIL {
		t.Fatalf("env PATH must not leak the real environment, got: %s", res.Inspect())
	}
}

// ---- network whitelist ----

func TestNetworkWhitelistHostMatching(t *testing.T) {
	p := NewSandboxProfile("wl")
	p.Network = true
	p.NetworkWhitelist = []string{"api.github.com"}

	if err := p.CanNetworkTo("https://api.github.com/repos/MachuraHarry/pipe"); err != nil {
		t.Fatalf("exact host should match: %v", err)
	}
	if err := p.CanNetworkTo("https://sub.api.github.com/x"); err != nil {
		t.Fatalf("subdomain should match: %v", err)
	}
	if err := p.CanNetworkTo("https://notapi.github.com/x"); err == nil {
		t.Fatal("notapi.github.com must NOT match")
	}
	if err := p.CanNetworkTo("https://attacker.com/?next=api.github.com"); err == nil {
		t.Fatal("query-embedded host must NOT match")
	}
}

func TestNetworkWhitelistPort(t *testing.T) {
	p := NewSandboxProfile("wlport")
	p.Network = true
	p.NetworkWhitelist = []string{"api.github.com:443"}

	if err := p.CanNetworkTo("https://api.github.com/x"); err != nil {
		t.Fatalf("https default port 443 should match: %v", err)
	}
	if err := p.CanNetworkTo("http://api.github.com/x"); err == nil {
		t.Fatal("http port 80 must not match a :443 whitelist entry")
	}
}

func TestHTTPRedirectWhitelistEnforced(t *testing.T) {
	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("internal-secret"))
	}))
	defer srvB.Close()

	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srvB.URL, http.StatusFound)
	}))
	defer srvA.Close()

	hostA := strings.TrimPrefix(srvA.URL, "http://")

	p := NewSandboxProfile("redir")
	p.Network = true
	p.NetworkWhitelist = []string{hostA}
	defer withProfile(p)()

	res := bHttpGet(&String{Value: srvA.URL})
	if res.Type() != ERROR {
		t.Fatalf("redirect to a non-whitelisted host must be blocked, got: %s", res.Inspect())
	}
}

// ---- locking ----

func TestSetSandboxNoneBlockedWhenLocked(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)
	prevLocked := sandboxStartLocked
	defer func() { sandboxStartLocked = prevLocked }()
	sandboxStartLocked = true

	res := bSetSandbox(&String{Value: "none"})
	if res.Type() != ERROR {
		t.Fatalf("set_sandbox \"none\" must be blocked when the sandbox is locked, got: %s", res.Inspect())
	}
}

func TestWithSandboxNoneBlockedWhenLocked(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)
	prevLocked := sandboxStartLocked
	defer func() { sandboxStartLocked = prevLocked }()
	sandboxStartLocked = true

	res := bWithSandbox(&String{Value: "none"}, &BuiltinInfo{Name: "noop"})
	if res.Type() != ERROR {
		t.Fatalf("with_sandbox \"none\" must be blocked when the sandbox is locked, got: %s", res.Inspect())
	}
}

func TestLockedProfileCannotSwitch(t *testing.T) {
	p := NewSandboxProfile("cell")
	p.FSAccess = FSNone
	p.PathPolicy = &defaultPathPolicy{access: FSNone}
	p.SetLocked()
	defer withProfile(p)()

	res := bSetSandbox(&String{Value: "noexec"})
	if res.Type() != ERROR {
		t.Fatalf("set_sandbox must be blocked from a locked profile, got: %s", res.Inspect())
	}
}

// ---- budget ----

func TestBudgetPreCheck(t *testing.T) {
	p := NewSandboxProfile("budget")
	p.AI = true
	p.Budget = 0.0000001
	if err := p.CanAI(); err == nil {
		t.Fatal("tiny budget must block a call before it is issued")
	}

	p.Budget = 100
	if err := p.CanAI(); err != nil {
		t.Fatalf("large budget should allow a call: %v", err)
	}
}

// ---- sandboxed tool execution ----

func TestWithActiveProfileAppliesSnapshot(t *testing.T) {
	p := NewSandboxProfile("toolbox")
	p.FSAccess = FSNone
	p.PathPolicy = &defaultPathPolicy{access: FSNone}

	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	var observed string
	withActiveProfile(p, func() Object {
		observed = ActiveProfile.Load().Name
		return NILOBJ
	})
	if observed != "toolbox" {
		t.Fatalf("tool body saw profile %q, want toolbox", observed)
	}
	if ActiveProfile.Load() != prev {
		t.Fatal("active profile was not restored after withActiveProfile")
	}
}

// TestToolExecutorDoesNotGateOnExec verifies that ai_with_tools stays usable
// under a profile with exec disabled: tools run and are audited, while the
// profile's own gates still block the dangerous builtins.
func TestToolExecutorDoesNotGateOnExec(t *testing.T) {
	p, dir := tempOnlyProfile(t)
	p.Exec = false
	p.AI = true
	p.AuditLog = true
	RegisterProfile(p.Name, p)
	defer withProfile(p)()

	toolRegistry["rt_write"] = ToolEntry{Fn: &BuiltinInfo{Name: "write_file", Fn: bWriteFile}}
	toolRegistry["rt_exec"] = ToolEntry{Fn: &BuiltinInfo{Name: "exec", Fn: bExec}}

	outside := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(outside, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := executeTool(p, "rt_write", map[string]interface{}{"path": outside, "content": "sneaky"})
	if err != nil {
		t.Fatalf("write_file tool must run under exec:false, got: %v", err)
	}
	if res == "" {
		t.Fatal("expected tool output")
	}
	data, _ := os.ReadFile(outside)
	if string(data) != "original" {
		t.Fatalf("outside file was modified through the tool: %q", data)
	}
	sandboxed := filepath.Join(dir, ".pipe_sandbox", "secret.txt")
	if got, err := os.ReadFile(sandboxed); err != nil || string(got) != "sneaky" {
		t.Fatalf("expected redirected sandbox copy, got %q (err=%v)", got, err)
	}

	res, err = executeTool(p, "rt_exec", map[string]interface{}{"command": "id"})
	if err != nil {
		t.Fatalf("builtin failures surface as tool output, not as executor errors: %v", err)
	}
	if !strings.Contains(res, "E_SANDBOX") {
		t.Fatalf("exec tool must be blocked by the profile's exec gate, got: %q", res)
	}

	if got := len(p.GetAuditLog()); got != 2 {
		t.Fatalf("expected 2 audit entries, got %d", got)
	}

	delete(toolRegistry, "rt_write")
	delete(toolRegistry, "rt_exec")
}

func TestToolExecutorMaxCallsEnforced(t *testing.T) {
	p := NewSandboxProfile("rtmax")
	p.FSAccess = FSNone
	p.PathPolicy = &defaultPathPolicy{access: FSNone}
	p.MaxToolCalls = 1
	p.AuditLog = true
	defer withProfile(p)()

	toolRegistry["rt_noop"] = ToolEntry{Fn: &BuiltinInfo{Name: "noop", Fn: func(args ...Object) Object { return NILOBJ }}}

	if _, err := executeTool(p, "rt_noop", nil); err != nil {
		t.Fatalf("first tool call should be allowed: %v", err)
	}
	if _, err := executeTool(p, "rt_noop", nil); err == nil {
		t.Fatal("second tool call must be blocked by max_tool_calls")
	}

	delete(toolRegistry, "rt_noop")
}
