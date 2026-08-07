package object

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

// ---- IO ----

var PrintHook func(args ...Object)

var TryAIEvalFn func(source string) Object

var ScriptArgs []string

func init() {
	ai.SetCostHook(func(entry ai.CostEntry) {
		if ActiveProfile != nil {
			ActiveProfile.RecordCost(entry.CostUSD)
			if ActiveProfile.AuditLog {
				ActiveProfile.Audit("ai_call",
					fmt.Sprintf("provider=%s model=%s tokens=%d cost=%.6f cached=%v",
						entry.Provider, entry.Model, entry.TotalTokens, entry.CostUSD, entry.Cached))
			}
		}
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
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanRead(s.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	data, e := os.ReadFile(s.Value)
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
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	if e := os.WriteFile(p.Value, []byte(c.Value), 0644); e != nil {
		return err("write_file: " + e.Error())
	}
	return NILOBJ
}

func bEnv(args ...Object) Object {
	if len(args) != 1 {
		return err("env expects 1 argument (Name)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("env: Name must be a string")
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
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanExec(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowExec {
		return sandboxBlock("exec")
	}
	if len(args) != 1 {
		return err("exec expects 1 argument (command)")
	}
	cmd, ok := args[0].(*String)
	if !ok {
		return err("exec: command must be a string")
	}
	out, e := exec.Command("sh", "-c", cmd.Value).CombinedOutput()
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

