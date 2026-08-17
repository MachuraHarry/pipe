package object

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

// ---- IO ----

var PrintHook func(args ...Object)

var TryAIEvalFn func(source string) Object

var ScriptArgs []string

func init() {
	ai.SetCostHook(func(entry ai.CostEntry) {
		if ActiveProfile.Load() != nil {
			ActiveProfile.Load().RecordCost(entry.CostUSD)
			if ActiveProfile.Load().AuditLog {
				ActiveProfile.Load().Audit("ai_call",
					fmt.Sprintf("provider=%s model=%s tokens=%d cost=%.6f cached=%v",
						entry.Provider, entry.Model, entry.TotalTokens, entry.CostUSD, entry.Cached))
			}
		}
	})

	// Central sandbox gate: every AI-provider egress must pass before any
	// network call is made. This is the backstop that makes the round-3/4 bug
	// class (a builtin that forgot its own CanAI check) structurally impossible.
	ai.SetEgressGate(func(info ai.EgressInfo) error {
		p := ActiveProfile.Load()
		if p == nil {
			return nil
		}
		if p.Name == "none" {
			if !Sandbox.Enabled {
				return nil
			}
			switch info.Kind {
			case ai.EgressChat, ai.EgressStream, ai.EgressEmbed:
				if !Sandbox.AllowAI {
					return fmt.Errorf("E_SANDBOX: AI calls blocked by sandbox")
				}
			case ai.EgressSearch:
				if !Sandbox.AllowNet {
					return fmt.Errorf("E_SANDBOX: network access blocked by sandbox")
				}
			}
			return nil
		}
		switch info.Kind {
		case ai.EgressChat, ai.EgressStream, ai.EgressEmbed:
			return p.CanAI()
		case ai.EgressSearch:
			return p.CanNetwork()
		}
		return nil
	})
}

func bTryAIEval(args ...Object) Object {
	if len(args) < 1 {
		return &Error{Message: "_try_ai_eval expects 1 argument (source)"}
	}
	src, ok := args[0].(*String)
	if !ok {
		return &Error{Message: "_try_ai_eval: argument must be a string"}
	}
	if TryAIEvalFn == nil {
		return &Error{Message: "_try_ai_eval: not available (requires tree-walker)"}
	}
	return TryAIEvalFn(src.Value)
}

func bPrint(args ...Object) Object {
	if PrintHook != nil {
		PrintHook(args...)
		return NILOBJ
	}
	for _, arg := range args {
		fmt.Print(arg.Inspect())
		fmt.Print(" ")
	}
	fmt.Println()
	return NILOBJ
}

func bPrintRaw(args ...Object) Object {
	if PrintHook != nil {
		PrintHook(args...)
		return NILOBJ
	}
	for _, arg := range args {
		fmt.Print(arg.Inspect())
	}
	return NILOBJ
}

func bInput(args ...Object) Object {
	if len(args) > 0 {
		if prompt, ok := args[0].(*String); ok {
			fmt.Print(prompt.Value)
		}
	}
	var line string
	fmt.Scanln(&line)
	return &String{Value: line}
}

func bReadFile(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("read_file expects 1-2 arguments (path, mode?)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("read_file expects a string as path")
	}
	path := s.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		path, cerr = ActiveProfile.Load().canonicalRead(s.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return err("read_file: " + e.Error())
	}
	if len(args) == 2 {
		mode, ok := args[1].(*String)
		if ok && mode.Value == "bytes" {
			return &Bytes{Value: data}
		}
	}
	return &String{Value: string(data)}
}

func bWriteFile(args ...Object) Object {
	if len(args) != 2 {
		return err("write_file expects 2 arguments (path, content)")
	}
	p, ok := args[0].(*String)
	c, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("write_file: path and content must be strings")
	}
	path := p.Value
	if ActiveProfile.Load().Name != "none" {
		var cerr error
		path, cerr = ActiveProfile.Load().canonicalWrite(p.Value)
		if cerr != nil {
			return err(cerr.Error())
		}
	}
	if e := os.WriteFile(path, []byte(c.Value), 0644); e != nil {
		return err("write_file: " + e.Error())
	}
	return NILOBJ
}

// secretEnvMarkers are substrings that identify secrets. Sandboxed code must
// never read these from the real process environment.
var secretEnvMarkers = []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "PASSWD", "CREDENTIAL", "APIKEY", "APISECRET"}

func blockedEnvName(name string) bool {
	upper := strings.ToUpper(name)
	for _, m := range secretEnvMarkers {
		if strings.Contains(upper, m) {
			return true
		}
	}
	return false
}

// bEnv implements `env(name)`. Under any active sandbox (a registered profile
// or the legacy --sandbox flag) the real process environment is never exposed:
// a profile only returns values from its explicit `Env` allowlist, and the flag
// path (which has no allowlist) masks everything. This is deterministic rather
// than name-heuristic: a secret whose name happens to avoid the marker
// substrings can no longer leak, because the value never comes from the host
// process environment.
func bEnv(args ...Object) Object {
	if len(args) != 1 {
		return err("env expects 1 argument (Name)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("env: Name must be a string")
	}
	if ActiveProfile.Load().Name != "none" {
		if blockedEnvName(name.Value) {
			return err("env: access to environment variable '" + name.Value + "' is blocked by sandbox policy")
		}
		val, exists := ActiveProfile.Load().Env[name.Value]
		if !exists {
			return NILOBJ
		}
		return &String{Value: val}
	}
	if Sandbox.Enabled {
		if blockedEnvName(name.Value) {
			return err("env: access to environment variable '" + name.Value + "' is blocked by sandbox policy")
		}
		return NILOBJ
	}
	val, exists := os.LookupEnv(name.Value)
	if !exists {
		return NILOBJ
	}
	return &String{Value: val}
}

func bSleep(args ...Object) Object {
	if len(args) != 1 {
		return err("sleep expects 1 argument (milliseconds)")
	}
	ms, ok := ToInt(args[0])
	if !ok {
		return err("sleep: milliseconds must be a number")
	}
	profile := ActiveProfile.Load()
	if profile.Name != "none" && profile.Timeout > 0 {
		maxMs := int64(profile.Timeout) * 1000
		if ms > maxMs {
			ms = maxMs
		}
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return NILOBJ
}

func bArgs(args ...Object) Object {
	elems := make([]Object, len(ScriptArgs))
	for i, a := range ScriptArgs {
		elems[i] = &String{Value: a}
	}
	return &List{Elements: elems}
}

func bReadStdin(args ...Object) Object {
	b, readErr := io.ReadAll(os.Stdin)
	if readErr != nil {
		return err("read_stdin: " + readErr.Error())
	}
	return &String{Value: string(b)}
}

func bExec(args ...Object) Object {
	if len(args) != 1 {
		return err("exec expects 1 argument (command)")
	}
	cmd, ok := args[0].(*String)
	if !ok {
		return err("exec: command must be a string")
	}
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanExecCommand(cmd.Value); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowExec {
		return sandboxBlock("exec")
	}
	profile := ActiveProfile.Load()
	shell, flag := execShell()
	var c *exec.Cmd
	if profile.Name != "none" && profile.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(profile.Timeout)*time.Second)
		defer cancel()
		c = exec.CommandContext(ctx, shell, flag, cmd.Value)
	} else {
		c = exec.Command(shell, flag, cmd.Value)
	}
	if profile.Name != "none" {
		env := os.Environ()
		for k, v := range profile.Env {
			env = append(env, k+"="+v)
		}
		c.Env = env
	}
	out, e := c.CombinedOutput()
	if e != nil {
		return &Map{Pairs: map[string]Object{
			"output": &String{Value: string(out)},
			"error":  &String{Value: e.Error()},
			"status": &Integer{Value: 1},
		}}
	}
	return &Map{Pairs: map[string]Object{
		"output": &String{Value: string(out)},
		"error":  &String{Value: ""},
		"status": &Integer{Value: 0},
	}}
}

// execShell returns the shell (and its run-command flag) used to execute
// `exec` commands. Unix uses sh -c, which is guaranteed by POSIX; Windows
// prefers cmd.exe /c, which is always present, so the sandboxed exec path
// stays functional without a bash installation.
func execShell() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", "/c"
	}
	return "sh", "-c"
}

func bDotenv(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("dotenv expects 1-2 arguments (file [, prefix])")
	}
	file, ok := args[0].(*String)
	if !ok {
		return err("dotenv: file must be a string")
	}
	prefix := ""
	if len(args) == 2 {
		if p, ok := args[1].(*String); ok {
			prefix = p.Value
		}
	}
	data, e := os.ReadFile(file.Value)
	if e != nil {
		return err("dotenv: " + e.Error())
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 1 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if prefix != "" {
			key = prefix + key
		}
		os.Setenv(key, val)
		count++
	}
	return &Integer{Value: int64(count)}
}
