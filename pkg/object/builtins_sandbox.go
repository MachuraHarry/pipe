package object

import (
	"strings"
	"time"
)

type SandboxConfig struct {
	Enabled   bool
	AllowAI   bool
	AllowExec bool
	AllowNet  bool
	AllowFS   bool
}

var Sandbox = SandboxConfig{}

func SetSandbox(enabled bool)          { Sandbox.Enabled = enabled }
func SetSandboxAllowAI(allowed bool)   { Sandbox.AllowAI = allowed }
func SetSandboxAllowExec(allowed bool) { Sandbox.AllowExec = allowed }
func SetSandboxAllowNet(allowed bool)  { Sandbox.AllowNet = allowed }
func SetSandboxAllowFS(allowed bool)   { Sandbox.AllowFS = allowed }

func sandboxBlock(feature string) *Error {
	msg := "SANDBOX: " + feature + " is disabled in sandbox mode"
	if strings.Contains(feature, "AI") || strings.Contains(feature, "ai") {
		msg += " — use --allow-ai or allow-ai: true to re-enable"
	}
	return &Error{Message: msg}
}

// ---- Sandbox Profile Builtins ----

func bSandboxProfile(args ...Object) Object {
	if len(args) < 2 {
		return err("sandbox_profile needs name and config block")
	}

	name, ok := args[0].(*String)
	if !ok {
		return err("sandbox_profile name must be a string")
	}

	config, ok := args[1].(*Map)
	if !ok {
		return err("sandbox_profile config must be a block/map")
	}

	profile := NewSandboxProfile(name.Value)

	for key, val := range config.Pairs {
		switch key {
		case "fs":
			s, ok := val.(*String)
			if !ok {
				return err("sandbox_profile: fs must be a string")
			}
			fsLevel, fsErr := ParseFSAccess(s.Value)
			if fsErr != nil {
				return err(fsErr.Error())
			}
			profile.FSAccess = fsLevel

		case "network":
			b, ok := val.(*Boolean)
			if !ok {
				return err("sandbox_profile: network must be a bool")
			}
			profile.Network = b.Value

		case "exec":
			b, ok := val.(*Boolean)
			if !ok {
				return err("sandbox_profile: exec must be a bool")
			}
			profile.Exec = b.Value

		case "ai":
			b, ok := val.(*Boolean)
			if !ok {
				return err("sandbox_profile: ai must be a bool")
			}
			profile.AI = b.Value

		case "timeout":
			i, ok := val.(*Integer)
			if !ok {
				return err("sandbox_profile: timeout must be a number")
			}
			profile.Timeout = int(i.Value)

		case "env":
			m, ok := val.(*Map)
			if !ok {
				return err("sandbox_profile: env must be a map")
			}
			for ek, ev := range m.Pairs {
				if s, ok := ev.(*String); ok {
					profile.Env[ek] = s.Value
				}
			}

		case "work_dir":
			s, ok := val.(*String)
			if !ok {
				return err("sandbox_profile: work_dir must be a string")
			}
			profile.WorkDir = s.Value

		case "budget":
			switch v := val.(type) {
			case *Float:
				profile.Budget = v.Value
			case *Integer:
				profile.Budget = float64(v.Value)
			default:
				return err("sandbox_profile: budget must be a number")
			}

		case "network_whitelist":
			lst, ok := val.(*List)
			if !ok {
				return err("sandbox_profile: network_whitelist must be a list of strings")
			}
			profile.NetworkWhitelist = make([]string, 0, len(lst.Elements))
			for _, e := range lst.Elements {
				if s, ok := e.(*String); ok {
					profile.NetworkWhitelist = append(profile.NetworkWhitelist, s.Value)
				}
			}

		case "max_tool_calls":
			i, ok := val.(*Integer)
			if !ok {
				return err("sandbox_profile: max_tool_calls must be a number")
			}
			profile.MaxToolCalls = int(i.Value)

		case "audit_log":
			b, ok := val.(*Boolean)
			if !ok {
				return err("sandbox_profile: audit_log must be a bool")
			}
			profile.AuditLog = b.Value

		default:
			return err("sandbox_profile: unknown config key '" + key + "'")
		}
	}

	if regErr := RegisterProfile(name.Value, profile); regErr != nil {
		return err(regErr.Error())
	}

	return TRUE
}

func bSetSandbox(args ...Object) Object {
	if len(args) < 1 {
		return err("set_sandbox needs a profile name")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("set_sandbox name must be a string")
	}
	prof, profErr := GetProfile(name.Value)
	if profErr != nil {
		return err(profErr.Error())
	}
	ActiveProfile = prof
	return TRUE
}

func bWithSandbox(args ...Object) Object {
	if len(args) < 2 {
		return err("with_sandbox needs a profile name and a block/function")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("with_sandbox name must be a string")
	}

	prev := ActiveProfile
	defer func() { ActiveProfile = prev }()

	prof, profErr := GetProfile(name.Value)
	if profErr != nil {
		return err(profErr.Error())
	}
	ActiveProfile = prof

	switch fn := args[1].(type) {
	case *Function:
		if callUserFn != nil {
			return callUserFn(fn)
		}
		return err("with_sandbox: function execution not available")
	case *BuiltinInfo:
		return fn.Fn()
	default:
		return err("with_sandbox: second argument must be a function/block")
	}
}

func bAuditLog(args ...Object) Object {
	entries := ActiveProfile.GetAuditLog()
	elems := make([]Object, len(entries))
	for i, e := range entries {
		elems[i] = &Map{Pairs: map[string]Object{
			"time":    &String{Value: e.Time.Format(time.RFC3339)},
			"event":   &String{Value: e.Event},
			"detail":  &String{Value: e.Detail},
			"profile": &String{Value: e.Profile},
		}}
	}
	return &List{Elements: elems}
}

func bBudgetSpent(args ...Object) Object {
	spent := ActiveProfile.GetSpentBudget()
	return &Float{Value: spent}
}
