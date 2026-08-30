package object

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/ai"
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

// registerTestProfile registers a profile and schedules its removal from the
// global registry when the test ends, so repeated runs (-count=N) do not
// collide on the fixed profile name.
func registerTestProfile(t *testing.T, p *SandboxProfile) {
	t.Helper()
	if err := RegisterProfile(p.Name, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		profileRegistryMu.Lock()
		delete(profileRegistry, p.Name)
		profileRegistryMu.Unlock()
	})
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
	p := NewSandboxProfile("envblk")
	p.FSAccess = FSFull
	p.PathPolicy = &defaultPathPolicy{access: FSFull}
	p.Env = map[string]string{}
	defer withProfile(p)()

	res := bEnv(&String{Value: "OPENAI_API_KEY"})
	if res.Type() != ERROR {
		t.Fatalf("env of a secret variable must be blocked under a sandbox profile, got: %s", res.Inspect())
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

func TestEnvFlagPathMasksRealEnvironment(t *testing.T) {
	// The legacy --sandbox flag (no profile) must never expose the real
	// process environment, deterministically — even for names that do not
	// match the secret markers.
	t.Setenv("PIPE_LEAK_SENTINEL", "secret-value")
	t.Setenv("PIPE_PLAIN", "visible-nope")

	restore := withSandboxFlags(true, false)
	defer restore()

	if res := bEnv(&String{Value: "PIPE_LEAK_SENTINEL"}); res.Type() != NIL {
		t.Fatalf("env of a non-marker secret must be masked under --sandbox, got: %s", res.Inspect())
	}
	if res := bEnv(&String{Value: "PIPE_PLAIN"}); res.Type() != NIL {
		t.Fatalf("env of a non-marker var must be masked under --sandbox, got: %s", res.Inspect())
	}
	if res := bEnv(&String{Value: "DEEPSEEK_API_KEY"}); res.Type() != ERROR {
		t.Fatalf("env of a secret-marked var must error under --sandbox, got: %s", res.Inspect())
	}
}

func TestEnvNoSandboxReadsRealEnvironment(t *testing.T) {
	t.Setenv("PIPE_VISIBLE", "real-value")
	restore := withSandboxFlags(false, false)
	defer restore()

	res := bEnv(&String{Value: "PIPE_VISIBLE"})
	if s, ok := res.(*String); !ok || s.Value != "real-value" {
		t.Fatalf("env PIPE_VISIBLE = %s, want real-value when not sandboxed", res.Inspect())
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

// ---- subset / ratchet ----

func testProfile(name string, fs FSAccess, network, exec, ai bool, whitelist []string) *SandboxProfile {
	return &SandboxProfile{
		Name:             name,
		FSAccess:         fs,
		Network:          network,
		Exec:             exec,
		AI:               ai,
		NetworkWhitelist: whitelist,
		Env:              map[string]string{},
		WorkDir:          ".",
		PathPolicy:       &defaultPathPolicy{access: fs},
	}
}

func TestIsSubsetOf(t *testing.T) {
	tests := []struct {
		name string
		sub  *SandboxProfile
		sup  *SandboxProfile
		want bool
	}{
		{"identical", testProfile("a", FSNone, false, false, false, nil), testProfile("a", FSNone, false, false, false, nil), true},
		{"fs subset", testProfile("a", FSReadOnly, false, false, false, nil), testProfile("a", FSFull, false, false, false, nil), true},
		{"fs superset rejected", testProfile("a", FSFull, false, false, false, nil), testProfile("a", FSReadOnly, false, false, false, nil), false},
		{"exec subset", testProfile("a", FSNone, false, false, false, nil), testProfile("a", FSNone, false, true, false, nil), true},
		{"exec superset rejected", testProfile("a", FSNone, false, true, false, nil), testProfile("a", FSNone, false, false, false, nil), false},
		{"network subset", testProfile("a", FSNone, false, false, false, nil), testProfile("a", FSNone, true, false, false, nil), true},
		{"network superset rejected", testProfile("a", FSNone, true, false, false, nil), testProfile("a", FSNone, false, false, false, nil), false},
		{"ai subset", testProfile("a", FSNone, false, false, false, nil), testProfile("a", FSNone, false, false, true, nil), true},
		{"ai superset rejected", testProfile("a", FSNone, false, false, true, nil), testProfile("a", FSNone, false, false, false, nil), false},
		{"empty whitelist is superset", testProfile("a", FSNone, true, false, false, []string{"evil.com"}), testProfile("a", FSNone, true, false, false, nil), true},
		{"empty sub whitelist is not subset of whitelisted super", testProfile("a", FSNone, true, false, false, nil), testProfile("a", FSNone, true, false, false, []string{"api.github.com"}), false},
		{"whitelist containment", testProfile("a", FSNone, true, false, false, []string{"api.github.com"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com", "openai.com"}), true},
		{"whitelist subdomain", testProfile("a", FSNone, true, false, false, []string{"sub.api.github.com"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com"}), true},
		{"whitelist extra entry rejected", testProfile("a", FSNone, true, false, false, []string{"api.github.com", "evil.com"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com"}), false},
		{"portless entry not subset of ported pattern", testProfile("a", FSNone, true, false, false, []string{"api.github.com"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com:443"}), false},
		{"same port is subset", testProfile("a", FSNone, true, false, false, []string{"api.github.com:443"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com:443"}), true},
		{"ported entry subset of portless pattern", testProfile("a", FSNone, true, false, false, []string{"api.github.com:443"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com"}), true},
		{"different port rejected", testProfile("a", FSNone, true, false, false, []string{"api.github.com:8080"}), testProfile("a", FSNone, true, false, false, []string{"api.github.com:443"}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sub.IsSubsetOf(tt.sup); got != tt.want {
				t.Fatalf("IsSubsetOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsSubsetOfLimits verifies that budget/max_tool_calls/timeout participate
// in the ratchet: an unlimited (0) or higher limit is more permissive and must
// not count as a subset.
func TestIsSubsetOfLimits(t *testing.T) {
	base := testProfile("base", FSNone, false, false, false, nil)
	base.Budget = 0.01
	base.MaxToolCalls = 3
	base.Timeout = 60

	limited := func(budget float64, calls, timeout int) *SandboxProfile {
		p := testProfile("sub", FSNone, false, false, false, nil)
		p.Budget = budget
		p.MaxToolCalls = calls
		p.Timeout = timeout
		return p
	}

	tests := []struct {
		name string
		sub  *SandboxProfile
		want bool
	}{
		{"identical limits", limited(0.01, 3, 60), true},
		{"lower budget", limited(0.005, 3, 60), true},
		{"higher budget", limited(0.02, 3, 60), false},
		{"unlimited budget", limited(0, 3, 60), false},
		{"lower max_tool_calls", limited(0.01, 2, 60), true},
		{"higher max_tool_calls", limited(0.01, 4, 60), false},
		{"unlimited max_tool_calls", limited(0.01, 0, 60), false},
		{"lower timeout", limited(0.01, 3, 30), true},
		{"higher timeout", limited(0.01, 3, 120), false},
		{"unlimited timeout", limited(0.01, 3, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sub.IsSubsetOf(base); got != tt.want {
				t.Fatalf("IsSubsetOf() = %v, want %v", got, tt.want)
			}
		})
	}

	// Two fully-unlimited profiles are equal and therefore subsets.
	both0 := testProfile("s", FSNone, false, false, false, nil)
	if !both0.IsSubsetOf(both0) {
		t.Fatal("two fully-unlimited profiles must be subsets of each other")
	}
}

// TestSetSandboxRatchetBlocksEscalation reproduces the escape: an agent locked
// into a restricted profile must not be able to switch to a pre-registered,
// more permissive profile — nor back to "none".
func TestSetSandboxRatchetBlocksEscalation(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	cage := testProfile("r-cage", FSNone, false, false, false, nil)
	registerTestProfile(t, cage)
	escape := testProfile("r-escape", FSFull, true, true, true, nil)
	registerTestProfile(t, escape)
	ActiveProfile.Store(cage)

	res := bSetSandbox(&String{Value: escape.Name})
	if res.Type() != ERROR {
		t.Fatalf("set_sandbox to a permissive profile must be ratcheted down, got: %s", res.Inspect())
	}
	if !strings.Contains(res.Inspect(), "ratchet down") {
		t.Fatalf("unexpected ratchet error: %s", res.Inspect())
	}
	if ActiveProfile.Load() != cage {
		t.Fatal("active profile must not change after a ratchet rejection")
	}

	res = bSetSandbox(&String{Value: "none"})
	if res.Type() != ERROR {
		t.Fatalf("set_sandbox \"none\" must be blocked by the full ratchet, got: %s", res.Inspect())
	}

	// A strictly more restrictive profile is still reachable.
	sub := testProfile("r-sub", FSNone, false, false, false, nil)
	registerTestProfile(t, sub)
	res = bSetSandbox(&String{Value: sub.Name})
	if res.Type() == ERROR {
		t.Fatalf("set_sandbox to a subset profile must be allowed, got: %s", res.Inspect())
	}
	if ActiveProfile.Load() != sub {
		t.Fatal("expected the subset profile to become active")
	}
}

// TestWithSandboxRatchetBlocksEscalation verifies the same guarantee for
// with_sandbox.
func TestWithSandboxRatchetBlocksEscalation(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	cage := testProfile("w-cage", FSNone, false, false, false, nil)
	registerTestProfile(t, cage)
	escape := testProfile("w-escape", FSFull, true, true, true, nil)
	registerTestProfile(t, escape)
	ActiveProfile.Store(cage)

	res := bWithSandbox(&String{Value: escape.Name}, &BuiltinInfo{Name: "noop", Fn: func(args ...Object) Object { return NILOBJ }})
	if res.Type() != ERROR {
		t.Fatalf("with_sandbox to a permissive profile must be ratcheted down, got: %s", res.Inspect())
	}
	if ActiveProfile.Load() != cage {
		t.Fatal("active profile must not change after a ratchet rejection")
	}
}

// TestSandboxProfileRegistrationRatchet verifies that a restricted profile
// cannot be used to register a more permissive one (the registration-level
// half of the escape).
func TestSandboxProfileRegistrationRatchet(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	cage := testProfile("reg-cage", FSReadOnly, false, false, false, nil)
	registerTestProfile(t, cage)
	t.Cleanup(func() {
		profileRegistryMu.Lock()
		delete(profileRegistry, "reg-ok")
		profileRegistryMu.Unlock()
	})
	ActiveProfile.Store(cage)

	permissive := &Map{Pairs: map[string]Object{
		"network": TRUE,
		"exec":    TRUE,
		"ai":      TRUE,
	}}
	res := bSandboxProfile(&String{Value: "reg-escape"}, permissive)
	if res.Type() != ERROR {
		t.Fatalf("registering a permissive profile under a restricted one must fail, got: %s", res.Inspect())
	}
	if _, err := GetProfile("reg-escape"); err == nil {
		t.Fatal("permissive profile must not be registered")
	}

	restricted := &Map{Pairs: map[string]Object{
		"fs":      &String{Value: "none"},
		"network": &Boolean{Value: false},
		"exec":    &Boolean{Value: false},
		"ai":      &Boolean{Value: false},
	}}
	res = bSandboxProfile(&String{Value: "reg-ok"}, restricted)
	if res.Type() == ERROR {
		t.Fatalf("registering a subset profile must be allowed, got: %s", res.Inspect())
	}
}

// ---- MCP client gates ----

func TestMcpUseStdioBlockedUnderNoExec(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	cage := testProfile("mcp-exec", FSNone, false, false, false, nil)
	registerTestProfile(t, cage)
	ActiveProfile.Store(cage)

	res := bMcpUseStdio(&String{Value: "true"})
	if res.Type() != ERROR {
		t.Fatalf("mcp_use_stdio must be blocked under exec:false, got: %s", res.Inspect())
	}
	if !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected an E_SANDBOX error, got: %s", res.Inspect())
	}

	// Under the "none" profile the gate is skipped; the error must come from
	// the failed spawn instead, proving the subprocess was actually attempted.
	ActiveProfile.Store(prev)
	res = bMcpUseStdio(&String{Value: "pipe-no-such-binary-xyz"})
	if res.Type() != ERROR {
		t.Fatalf("expected a spawn failure, got: %s", res.Inspect())
	}
	if strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("no sandbox gate must apply under profile 'none', got: %s", res.Inspect())
	}
}

func jsonRPCErrorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID any `json:"id"`
		}
		id := 0
		if json.Unmarshal(body, &req) == nil {
			switch v := req.ID.(type) {
			case float64:
				id = int(v)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"error":{"code":-32000,"message":"boom"}}`, id)
	}))
}

func TestMcpUseSSEBlockedUnderNoNetwork(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	ts := jsonRPCErrorServer()
	defer ts.Close()

	cage := testProfile("mcp-net", FSNone, false, false, false, nil)
	registerTestProfile(t, cage)
	ActiveProfile.Store(cage)

	res := bMcpUseSSE(&String{Value: ts.URL})
	if res.Type() != ERROR {
		t.Fatalf("mcp_use_sse must be blocked under network:false, got: %s", res.Inspect())
	}
	if !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected an E_SANDBOX error, got: %s", res.Inspect())
	}
}

func TestMcpUseSSEBlockedByWhitelist(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	ts := jsonRPCErrorServer()
	defer ts.Close()

	wl := testProfile("mcp-wl", FSNone, true, false, false, []string{"allowed.example"})
	registerTestProfile(t, wl)
	ActiveProfile.Store(wl)

	res := bMcpUseSSE(&String{Value: ts.URL})
	if res.Type() != ERROR {
		t.Fatalf("mcp_use_sse to a non-whitelisted host must be blocked, got: %s", res.Inspect())
	}
	if !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("expected an E_SANDBOX error, got: %s", res.Inspect())
	}
}

func TestMcpUseSSEAllowedByWhitelist(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	ts := jsonRPCErrorServer()
	defer ts.Close()

	host := strings.TrimPrefix(ts.URL, "http://")
	wl := testProfile("mcp-ok", FSNone, true, false, false, []string{host})
	registerTestProfile(t, wl)
	ActiveProfile.Store(wl)

	res := bMcpUseSSE(&String{Value: ts.URL})
	if res.Type() != ERROR {
		t.Fatalf("expected the MCP handshake to fail, got: %s", res.Inspect())
	}
	if strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("whitelisted host must pass the gate, got: %s", res.Inspect())
	}
	if !strings.Contains(res.Inspect(), "boom") {
		t.Fatalf("expected the server's rpc error to surface, got: %s", res.Inspect())
	}
}

// TestMcpUseSSERedirectGate verifies that a whitelisted endpoint cannot be
// used as a redirect pivot to a non-whitelisted host.
func TestMcpUseSSERedirectGate(t *testing.T) {
	prev := ActiveProfile.Load()
	defer ActiveProfile.Store(prev)

	tsB := jsonRPCErrorServer()
	defer tsB.Close()
	tsA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tsB.URL, http.StatusFound)
	}))
	defer tsA.Close()

	hostA := strings.TrimPrefix(tsA.URL, "http://")
	wl := testProfile("mcp-redir", FSNone, true, false, false, []string{hostA})
	registerTestProfile(t, wl)
	ActiveProfile.Store(wl)

	res := bMcpUseSSE(&String{Value: tsA.URL})
	if res.Type() != ERROR {
		t.Fatalf("expected the redirect to be blocked, got: %s", res.Inspect())
	}
	if !strings.Contains(res.Inspect(), "E_SANDBOX") {
		t.Fatalf("redirect target must be checked against the whitelist, got: %s", res.Inspect())
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
	registerTestProfile(t, p)
	defer withProfile(p)()

	toolRegistry["rt_write"] = ToolEntry{
		Def: ai.ToolDef{Name: "write_file", Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{"path", "content"},
		}},
		Fn: &BuiltinInfo{Name: "write_file", Fn: bWriteFile},
	}
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

// ---- exec whitelist ----

// TestSplitShellWords covers the tokenizer that both CanExecCommand (the
// whitelist gate) and buildExecCommand (the actual argv-mode invocation) use.
// It intentionally gives no meaning to shell metacharacters (&&, |, $(...),
// env-assignment prefixes, cd) — see TestExecWhitelistBlocksShellInjection
// for why that's the point, not a gap.
func TestSplitShellWords(t *testing.T) {
	cases := []struct {
		cmd  string
		want []string
	}{
		{"git diff", []string{"git", "diff"}},
		{"git log -1 --format=%H -- file", []string{"git", "log", "-1", "--format=%H", "--", "file"}},
		{"/usr/bin/git rev-parse --short HEAD", []string{"/usr/bin/git", "rev-parse", "--short", "HEAD"}},
		{"'git' status", []string{"git", "status"}},
		{`git -C "D:\a\_temp\fixture" diff --numstat`, []string{"git", "-C", `D:\a\_temp\fixture`, "diff", "--numstat"}},
		{"echo hi | grep x", []string{"echo", "hi", "|", "grep", "x"}},
		{"cd repo && git status", []string{"cd", "repo", "&&", "git", "status"}},
		{"FOO=bar GIT_PAGER= git log", []string{"FOO=bar", "GIT_PAGER=", "git", "log"}},
		{"", nil},
	}
	for _, c := range cases {
		got, err := splitShellWords(c.cmd)
		if err != nil {
			t.Errorf("splitShellWords(%q): unexpected error: %v", c.cmd, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("splitShellWords(%q) = %v, want %v", c.cmd, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("splitShellWords(%q) = %v, want %v", c.cmd, got, c.want)
				break
			}
		}
	}

	if _, err := splitShellWords(`echo "unterminated`); err == nil {
		t.Error("splitShellWords: expected an error for an unterminated double quote")
	}
	if _, err := splitShellWords(`echo 'unterminated`); err == nil {
		t.Error("splitShellWords: expected an error for an unterminated single quote")
	}
}

// TestExecWhitelistBlocksShellInjection is the round-9 audit regression test:
// exec_whitelist must not be defeatable by appending shell syntax to an
// otherwise-whitelisted command. Before the fix, "echo" being whitelisted let
// "echo hi && echo INJECTED" reach a real shell (sh -c) unfiltered, running
// INJECTED as its own process. After the fix, the whole string is tokenized
// and run as literal argv to "echo" — no shell — so "&&"/"echo"/"INJECTED"
// are just three more arguments echo prints back, not shell syntax.
func TestExecWhitelistBlocksShellInjection(t *testing.T) {
	p := NewSandboxProfile("exw-inject")
	p.Exec = true
	p.ExecWhitelist = []string{"echo"}
	defer withProfile(p)()

	cases := []struct {
		name string
		cmd  string
		want string
	}{
		{"chain operator", "echo hi && echo INJECTED", "hi && echo INJECTED\n"},
		{"command substitution", "echo $(whoami)", "$(whoami)\n"},
		{"semicolon", "echo hi; echo INJECTED", "hi; echo INJECTED\n"},
		{"backtick", "echo `id`", "`id`\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := bExec(&String{Value: c.cmd})
			m, ok := result.(*Map)
			if !ok {
				t.Fatalf("exec(%q): expected a result map, got %v", c.cmd, result)
			}
			out, ok := m.Pairs["output"].(*String)
			if !ok {
				t.Fatalf("exec(%q): result map has no string output", c.cmd)
			}
			if out.Value != c.want {
				t.Fatalf("exec(%q) output = %q, want literal %q (no shell interpretation)", c.cmd, out.Value, c.want)
			}
		})
	}
}

// TestExecWhitelistNoLongerSupportsShellComposition documents an intentional
// behavior change from the round-9 fix: exec_whitelist no longer recognizes a
// binary buried after cd/&&/env-assignment prefixes, because doing so
// required parsing shell syntax that the fix deliberately stops interpreting.
// A single simple command is exec_whitelist's supported shape now.
func TestExecWhitelistNoLongerSupportsShellComposition(t *testing.T) {
	p := NewSandboxProfile("exw-nocomposition")
	p.Exec = true
	p.ExecWhitelist = []string{"git"}
	defer withProfile(p)()

	if err := p.CanExecCommand("cd /tmp && git status"); err == nil {
		t.Fatal("'cd ... && git ...' must be rejected: argv[0] is 'cd', not 'git'")
	}
	if err := p.CanExecCommand("FOO=bar GIT_PAGER= git log"); err == nil {
		t.Fatal("leading env-assignment prefixes must be rejected: argv[0] is 'FOO=bar', not 'git'")
	}
}

func TestExecShellFlags(t *testing.T) {
	shell, flag := execShell()
	if runtime.GOOS == "windows" {
		if shell != "cmd.exe" || flag != "/c" {
			t.Fatalf("windows execShell = %q %q, want cmd.exe /c", shell, flag)
		}
	} else {
		if shell != "sh" || flag != "-c" {
			t.Fatalf("unix execShell = %q %q, want sh -c", shell, flag)
		}
	}
}

func TestExecWhitelistAllowsGitBlocksOthers(t *testing.T) {
	p := NewSandboxProfile("exw")
	p.Exec = true
	p.ExecWhitelist = []string{"git"}
	defer withProfile(p)()

	if err := p.CanExecCommand("git diff --numstat"); err != nil {
		t.Fatalf("git must be allowed: %v", err)
	}
	// "cd /tmp && git status" is intentionally no longer allowed — see
	// TestExecWhitelistNoLongerSupportsShellComposition.
	if err := p.CanExecCommand("curl http://example.com"); err == nil {
		t.Fatal("curl must be blocked by exec whitelist")
	}
	if err := p.CanExecCommand("sh -c 'echo hi'"); err == nil {
		t.Fatal("sh must be blocked by exec whitelist")
	}
	if err := p.CanExecCommand(""); err == nil {
		t.Fatal("empty command must be rejected")
	}
}

func TestExecWhitelistEmptyMeansAllowAll(t *testing.T) {
	p := NewSandboxProfile("exw-all")
	p.Exec = true
	defer withProfile(p)()

	if err := p.CanExecCommand("anything --goes"); err != nil {
		t.Fatalf("empty whitelist must allow all: %v", err)
	}
}

func TestExecWhitelistSubsetRatchet(t *testing.T) {
	all := NewSandboxProfile("exw-super")
	all.Exec = true

	gitOnly := NewSandboxProfile("exw-sub")
	gitOnly.Exec = true
	gitOnly.ExecWhitelist = []string{"git"}

	curlOnly := NewSandboxProfile("exw-curl")
	curlOnly.Exec = true
	curlOnly.ExecWhitelist = []string{"curl"}

	gitAndCurl := NewSandboxProfile("exw-both")
	gitAndCurl.Exec = true
	gitAndCurl.ExecWhitelist = []string{"git", "curl"}

	if !gitOnly.IsSubsetOf(all) {
		t.Error("git-only must be a subset of allow-all")
	}
	if !curlOnly.IsSubsetOf(gitAndCurl) {
		t.Error("curl-only must be a subset of git+curl")
	}
	if gitOnly.IsSubsetOf(curlOnly) {
		t.Error("git-only must NOT be a subset of curl-only")
	}
	if gitAndCurl.IsSubsetOf(gitOnly) {
		t.Error("git+curl must NOT be a subset of git-only")
	}
	if all.IsSubsetOf(gitOnly) {
		t.Error("allow-all must NOT be a subset of git-only")
	}
}

func TestExecWhitelistBlocksViaProfileBuiltin(t *testing.T) {
	// Register the profile through bSandboxProfile and verify a stray exec is
	// rejected while git is accepted, mirroring the fluentloop use case.
	config := &Map{Pairs: map[string]Object{
		"fs":             &String{Value: "read-only"},
		"exec":           TRUE,
		"exec_whitelist": &List{Elements: []Object{&String{Value: "git"}}},
		"network":        FALSE,
		"ai":             FALSE,
	}}
	if res := bSandboxProfile(&String{Value: "exw-builtin"}, config); res.Type() == ERROR {
		t.Fatalf("sandbox_profile rejected: %s", res.Inspect())
	}

	p, _ := GetProfile("exw-builtin")
	t.Cleanup(func() {
		profileRegistryMu.Lock()
		delete(profileRegistry, "exw-builtin")
		profileRegistryMu.Unlock()
	})
	if err := p.CanExecCommand("git rev-parse --short HEAD"); err != nil {
		t.Fatalf("git must be allowed in exw-builtin: %v", err)
	}
	if err := p.CanExecCommand("curl http://x"); err == nil {
		t.Fatal("curl must be blocked in exw-builtin")
	}
}
