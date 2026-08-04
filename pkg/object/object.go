package object

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MachuraHarry/pipe/pkg/ai"
	"github.com/MachuraHarry/pipe/pkg/ast"
)

type ObjectType string

const (
	INTEGER           ObjectType = "INTEGER"
	FLOAT                        = "FLOAT"
	STRING                       = "STRING"
	BOOLEAN                      = "BOOLEAN"
	NIL                          = "NIL"
	FUNCTION                     = "FUNCTION"
	COMPILED_FUNCTION            = "COMPILED_FUNCTION"
	CLOSURE                      = "CLOSURE"
	LIST                         = "LIST"
	MAP                          = "MAP"
	FUTURE                       = "FUTURE"
	ERROR                        = "ERROR"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct{ Value int64 }

func (i *Integer) Type() ObjectType { return INTEGER }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Float struct{ Value float64 }

func (f *Float) Type() ObjectType { return FLOAT }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

type String struct{ Value string }

func (s *String) Type() ObjectType { return STRING }
func (s *String) Inspect() string  { return s.Value }

type Boolean struct{ Value bool }

func (b *Boolean) Type() ObjectType { return BOOLEAN }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type NilObject struct{}

func (n *NilObject) Type() ObjectType { return NIL }
func (n *NilObject) Inspect() string  { return "nil" }

type Function struct {
	Name       string
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
	EvalCtx    interface{} // *eval.EvalContext, avoids circular import
}

func (f *Function) Type() ObjectType { return FUNCTION }
func (f *Function) Inspect() string {
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.Value)
	}
	return fmt.Sprintf("fn(%s)", strings.Join(params, ", "))
}

type List struct{ Elements []Object }

func (l *List) Type() ObjectType { return LIST }
func (l *List) Inspect() string {
	elems := []string{}
	for _, e := range l.Elements {
		elems = append(elems, e.Inspect())
	}
	return fmt.Sprintf("[%s]", strings.Join(elems, ", "))
}

type Map struct{ Pairs map[string]Object }

func (m *Map) Type() ObjectType { return MAP }
func (m *Map) Inspect() string {
	pairs := []string{}
	for k, v := range m.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, v.Inspect()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

type Error struct{ Message string }

func (e *Error) Type() ObjectType { return ERROR }
func (e *Error) Inspect() string  { return e.Message }

type Result struct {
	Ok  bool
	Val Object
	Err string
}

func (r *Result) Type() ObjectType { return "RESULT" }
func (r *Result) Inspect() string {
	if r.Ok {
		return "Ok(" + r.Val.Inspect() + ")"
	}
	return "Err(" + r.Err + ")"
}

type Future struct {
	Val  Object
	Done chan struct{}
}

func NewFuture() *Future {
	return &Future{Done: make(chan struct{})}
}

func (f *Future) Type() ObjectType { return FUTURE }
func (f *Future) Inspect() string {
	select {
	case <-f.Done:
		if f.Val != nil {
			return f.Val.Inspect()
		}
		return "Future(resolved: nil)"
	default:
		return "Future(pending)"
	}
}

func (f *Future) Resolve() Object {
	<-f.Done
	return f.Val
}

func EnsureResolved(obj Object) Object {
	if f, ok := obj.(*Future); ok {
		return f.Resolve()
	}
	return obj
}

type BuiltinInfo struct {
	Name string
	Fn   func(args ...Object) Object
}

func (bi *BuiltinInfo) Type() ObjectType { return "BUILTIN" }
func (bi *BuiltinInfo) Inspect() string  { return "builtin: " + bi.Name }

type CompiledFunction struct {
	Instructions interface{}
	NumLocals    int
	NumFree      int
}

type Closure struct {
	Fn   *CompiledFunction
	Free []Object
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION }
func (cf *CompiledFunction) Inspect() string  { return "compiled function" }
func (c *Closure) Type() ObjectType           { return CLOSURE }
func (c *Closure) Inspect() string            { return "closure" }

var (
	TRUE   = &Boolean{Value: true}
	FALSE  = &Boolean{Value: false}
	NILOBJ = &NilObject{}
)

func NativeBoolToBoolean(b bool) *Boolean {
	if b {
		return TRUE
	}
	return FALSE
}

func IsTruthy(obj Object) bool {
	switch obj {
	case NILOBJ:
		return false
	case FALSE:
		return false
	}
	return true
}

func ToFloat(obj Object) (float64, bool) {
	switch v := obj.(type) {
	case *Integer:
		return float64(v.Value), true
	case *Float:
		return v.Value, true
	case *String:
		f, err := strconv.ParseFloat(v.Value, 64)
		return f, err == nil
	}
	return 0, false
}

func ToInt(obj Object) (int64, bool) {
	switch v := obj.(type) {
	case *Integer:
		return v.Value, true
	case *Float:
		return int64(v.Value), true
	case *String:
		i, err := strconv.ParseInt(v.Value, 10, 64)
		return i, err == nil
	}
	return 0, false
}

func ValuesEqual(a, b Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a := a.(type) {
	case *Integer:
		return a.Value == b.(*Integer).Value
	case *Float:
		return a.Value == b.(*Float).Value
	case *String:
		return a.Value == b.(*String).Value
	case *Boolean:
		return a.Value == b.(*Boolean).Value
	}
	return false
}

// ---- Builtins ----

var Builtins = []BuiltinInfo{
	// IO / File System
	{"print", bPrint},
	{"input", bInput},
	{"exec", bExec},
	{"read_file", bReadFile},
	{"write_file", bWriteFile},
	{"append_file", bAppendFile},
	{"read_lines", bReadLines},
	{"file_exists", bFileExists},
	{"file_delete", bFileDelete},
	{"file_move", bFileMove},
	{"file_copy", bFileCopy},
	{"file_size", bFileSize},
	{"file_type", bFileType},
	{"list_dir", bListDir},
	{"make_dir", bMakeDir},
	{"remove_dir", bRemoveDir},
	{"path_join", bPathJoin},
	{"path_base", bPathBase},
	{"path_dir", bPathDir},
	{"path_ext", bPathExt},
	{"env", bEnv},
	{"sleep", bSleep},

	// Network
	{"http_get", bHttpGet},
	{"http_post", bHttpPost},
	{"http_get_json", bHttpGetJSON},
	{"parse_json", bParseJSON},
	{"to_json", bToJSON},

	// TCP
	{"tcp_listen", bTcpListen},
	{"tcp_connect", bTcpConnect},
	{"tcp_accept", bTcpAccept},
	{"tcp_read", bTcpRead},
	{"tcp_write", bTcpWrite},
	{"tcp_close", bTcpClose},

	// String
	{"upper", bUpper},
	{"lower", bLower},
	{"trim", bTrim},
	{"split", bSplit},
	{"join", bJoin},
	{"contains", bContains},

	// List
	{"len", bLen},
	{"push", bPush},
	{"pop", bPop},
	{"at", bAt},
	{"slice_list", bSliceList},
	{"sort", bSort},
	{"range", bRange},

	// List — higher order (all modes, including inline lambdas)
	{"map", bMap},
	{"filter", bFilter},
	{"reduce", bReduce},
	{"each", bEach},

	// Math
	{"abs", bAbs},
	{"min", bMin},
	{"max", bMax},
	{"pow", bPow},
	{"sqrt", bSqrt},
	{"round", bRound},

	// Map
	{"keys", bKeys},
	{"values", bValues},
	{"get", bGet},
	{"set", bSet},

	// Type checks
	{"type_of", bTypeOf},
	{"is_num", bIsNum},
	{"is_str", bIsStr},
	{"is_list", bIsList},
	{"is_map", bIsMap},
	{"is_nil", bIsNil},

	// Conversion
	{"to_str", bToStr},
	{"to_num", bToNum},

	// Result type
	{"Ok", bOk},
	{"Err", bErr},
	{"is_ok", bIsOk},
	{"is_err", bIsErr},
	{"unwrap", bUnwrap},
	{"unwrap_or", bUnwrapOr},

	// Encoding
	{"base64_encode", bBase64Encode},
	{"base64_decode", bBase64Decode},

	// Regex
	{"regex_match", bRegexMatch},
	{"regex_replace", bRegexReplace},

	// Date/Time
	{"now", bNow},
	{"format_time", bFormatTime},

	// Random
	{"random", bRandom},
	{"random_range", bRandomRange},

	// AI — Configuration
	{"ai_provider", bAiProvider},
	{"ai_model", bAiModel},
	{"ai_timeout", bAiTimeout},
	{"ai_host", bAiHost},
	{"ai_cache", bAiCache},

	// AI — Retrieval
	{"web_search", bWebSearch},

	// AI — Low-level Chat
	{"ai_chat", bAiChat},
	{"ai_chat_json", bAiChatJSON},

	// AI — High-level Convenience
	{"summarize", bSummarize},
	{"translate", bTranslate},
	{"classify", bClassify},
	{"extract", bExtract},
	{"generate", bGenerate},
	{"generate_json", bGenerateJSON},
	{"ask", bAsk},

	// AI — Streaming
	{"ai_stream", bAiStream},

	// AI — Parallel
	{"ai_parallel", bAiParallel},
	{"ai_batch", bAiBatch},
	{"ai_rate_limit", bAiRateLimit},

	// AI — Embeddings
	{"embed", bEmbed},
	{"embed_batch", bEmbedBatch},
	{"cosine_sim", bCosineSim},
	{"dot_product", bDotProduct},
	{"nearest", bNearest},

	// Sandbox Profiles
	{"sandbox_profile", bSandboxProfile},
	{"set_sandbox", bSetSandbox},
	{"with_sandbox", bWithSandbox},

	// AI — Tool Calling
	{"ai_tool", bAiTool},
	{"ai_with_tools", bAiWithTools},

	// AI — Agents
	{"agent", bAgent},
	{"agent_ask", bAgentAsk},
	{"agent_clear", bAgentClear},

	// Test Assertions
	{"assert", bAssert},
	{"assert_eq", bAssertEq},
	{"assert_not_eq", bAssertNotEq},
	{"assert_lt", bAssertLt},
	{"assert_gt", bAssertGt},
	{"assert_error", bAssertError},
}

// ---- IO ----

var PrintHook func(args ...Object)

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
	if len(args) != 1 {
		return err("read_file expects 1 argument (path)")
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
	return &String{Value: string(data)}
}

func bWriteFile(args ...Object) Object {
	if len(args) != 2 {
		return err("write_file expects 2 arguments (path, content)")
	}
	p, ok := args[0].(*String)
	c, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("write_file: path und content must be strings")
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
	val := os.Getenv(name.Value)
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

// ---- File System ----

func bAppendFile(args ...Object) Object {
	if len(args) != 2 {
		return err("append_file expects 2 arguments (path, content)")
	}
	p, ok := args[0].(*String)
	c, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("append_file: path und content must be strings")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	f, e := os.OpenFile(p.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return err("append_file: " + e.Error())
	}
	defer f.Close()
	if _, e := f.WriteString(c.Value); e != nil {
		return err("append_file: " + e.Error())
	}
	return NILOBJ
}

func bReadLines(args ...Object) Object {
	if len(args) != 1 {
		return err("read_lines expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("read_lines: path must be a string")
	}
	data, e := os.ReadFile(p.Value)
	if e != nil {
		return err("read_lines: " + e.Error())
	}
	lines := strings.Split(string(data), "\n")
	elems := make([]Object, len(lines))
	for i, l := range lines {
		elems[i] = &String{Value: l}
	}
	return &List{Elements: elems}
}

func bFileExists(args ...Object) Object {
	if len(args) != 1 {
		return err("file_exists expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_exists: path must be a string")
	}
	_, e := os.Stat(p.Value)
	return NativeBoolToBoolean(e == nil)
}

func bFileDelete(args ...Object) Object {
	if len(args) != 1 {
		return err("file_delete expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_delete: path must be a string")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowFS {
		return sandboxBlock("file_delete (filesystem write)")
	}
	if e := os.Remove(p.Value); e != nil {
		return err("file_delete: " + e.Error())
	}
	return NILOBJ
}

func bFileMove(args ...Object) Object {
	if len(args) != 2 {
		return err("file_move expects 2 arguments (source, destination)")
	}
	src, ok := args[0].(*String)
	dst, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("file_move: paths must be strings")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(dst.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	if e := os.Rename(src.Value, dst.Value); e != nil {
		return err("file_move: " + e.Error())
	}
	return NILOBJ
}

func bFileCopy(args ...Object) Object {
	if len(args) != 2 {
		return err("file_copy expects 2 arguments (source, destination)")
	}
	src, ok := args[0].(*String)
	dst, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("file_copy: paths must be strings")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanRead(src.Value); canErr != nil {
			return err(canErr.Error())
		}
		if canErr := ActiveProfile.CanWrite(dst.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	srcFile, e := os.Open(src.Value)
	if e != nil {
		return err("file_copy: " + e.Error())
	}
	defer srcFile.Close()
	dstFile, e := os.Create(dst.Value)
	if e != nil {
		return err("file_copy: " + e.Error())
	}
	defer dstFile.Close()
	if _, e := io.Copy(dstFile, srcFile); e != nil {
		return err("file_copy: " + e.Error())
	}
	return NILOBJ
}

func bFileSize(args ...Object) Object {
	if len(args) != 1 {
		return err("file_size expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_size: path must be a string")
	}
	info, e := os.Stat(p.Value)
	if e != nil {
		return err("file_size: " + e.Error())
	}
	return &Integer{Value: info.Size()}
}

func bFileType(args ...Object) Object {
	if len(args) != 1 {
		return err("file_type expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_type: path must be a string")
	}
	info, e := os.Stat(p.Value)
	if e != nil {
		return err("file_type: " + e.Error())
	}
	if info.IsDir() {
		return &String{Value: "dir"}
	}
	return &String{Value: "file"}
}

func bListDir(args ...Object) Object {
	path := "."
	if len(args) >= 1 {
		p, ok := args[0].(*String)
		if !ok {
			return err("list_dir: path must be a string")
		}
		path = p.Value
	}
	entries, e := os.ReadDir(path)
	if e != nil {
		return err("list_dir: " + e.Error())
	}
	elems := make([]Object, len(entries))
	for i, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		elems[i] = &String{Value: name}
	}
	sort.Slice(elems, func(i, j int) bool {
		return elems[i].(*String).Value < elems[j].(*String).Value
	})
	return &List{Elements: elems}
}

func bMakeDir(args ...Object) Object {
	if len(args) != 1 {
		return err("make_dir expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("make_dir: path must be a string")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	}
	if e := os.MkdirAll(p.Value, 0755); e != nil {
		return err("make_dir: " + e.Error())
	}
	return NILOBJ
}

func bRemoveDir(args ...Object) Object {
	if len(args) != 1 {
		return err("remove_dir expects 1 argument (path)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("remove_dir: path must be a string")
	}
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanWrite(p.Value); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowFS {
		return sandboxBlock("remove_dir (filesystem write)")
	}
	if e := os.RemoveAll(p.Value); e != nil {
		return err("remove_dir: " + e.Error())
	}
	return NILOBJ
}

func bPathJoin(args ...Object) Object {
	parts := make([]string, len(args))
	for i, a := range args {
		s, ok := a.(*String)
		if !ok {
			return err("path_join: alle Argumente must be strings")
		}
		parts[i] = s.Value
	}
	return &String{Value: filepath.Join(parts...)}
}

func bPathBase(args ...Object) Object {
	if len(args) != 1 {
		return err("path_base expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_base expects a string")
	}
	return &String{Value: filepath.Base(s.Value)}
}

func bPathDir(args ...Object) Object {
	if len(args) != 1 {
		return err("path_dir expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_dir expects a string")
	}
	return &String{Value: filepath.Dir(s.Value)}
}

func bPathExt(args ...Object) Object {
	if len(args) != 1 {
		return err("path_ext expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_ext expects a string")
	}
	return &String{Value: filepath.Ext(s.Value)}
}

// ---- String ----

func bUpper(args ...Object) Object {
	if s, ok := strArg(args, "upper"); ok {
		return &String{Value: strings.ToUpper(s.Value)}
	}
	return err("upper expects a string")
}

func bLower(args ...Object) Object {
	if s, ok := strArg(args, "lower"); ok {
		return &String{Value: strings.ToLower(s.Value)}
	}
	return err("lower expects a string")
}

func bTrim(args ...Object) Object {
	if s, ok := strArg(args, "trim"); ok {
		return &String{Value: strings.TrimSpace(s.Value)}
	}
	return err("trim expects a string")
}

func bSplit(args ...Object) Object {
	if len(args) != 2 {
		return err("split expects 2 arguments")
	}
	s, ok := args[0].(*String)
	d, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("split: beide Argumente must be strings")
	}
	parts := strings.Split(s.Value, d.Value)
	elems := make([]Object, len(parts))
	for i, p := range parts {
		elems[i] = &String{Value: p}
	}
	return &List{Elements: elems}
}

func bJoin(args ...Object) Object {
	if len(args) != 2 {
		return err("join expects 2 arguments")
	}
	l, ok := args[0].(*List)
	d, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("join: list and string expected")
	}
	parts := make([]string, len(l.Elements))
	for i, e := range l.Elements {
		parts[i] = e.Inspect()
	}
	return &String{Value: strings.Join(parts, d.Value)}
}

func bContains(args ...Object) Object {
	if len(args) != 2 {
		return err("contains expects 2 arguments")
	}
	switch c := args[0].(type) {
	case *String:
		if sub, ok := args[1].(*String); ok {
			return NativeBoolToBoolean(strings.Contains(c.Value, sub.Value))
		}
		return err("contains: Substring must be a string")
	case *List:
		for _, e := range c.Elements {
			if ValuesEqual(e, args[1]) {
				return TRUE
			}
		}
		return FALSE
	}
	return err("contains expects string or list")
}

// ---- List ----

func bLen(args ...Object) Object {
	if len(args) != 1 {
		return err("len expects 1 argument")
	}
	switch a := args[0].(type) {
	case *String:
		return &Integer{Value: int64(len(a.Value))}
	case *List:
		return &Integer{Value: int64(len(a.Elements))}
	case *Map:
		return &Integer{Value: int64(len(a.Pairs))}
	}
	return err("len not supported")
}

func bPush(args ...Object) Object {
	if len(args) < 2 {
		return err("push expects at least 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("push: erstes Argument must be a list")
	}
	l.Elements = append(l.Elements, args[1:]...)
	return l
}

func bPop(args ...Object) Object {
	if len(args) != 1 {
		return err("pop expects 1 argument")
	}
	l, ok := args[0].(*List)
	if !ok || len(l.Elements) == 0 {
		return NILOBJ
	}
	last := l.Elements[len(l.Elements)-1]
	l.Elements = l.Elements[:len(l.Elements)-1]
	return last
}

func bAt(args ...Object) Object {
	if len(args) != 2 {
		return err("at expects 2 arguments")
	}
	idx, ok := ToInt(args[1])
	if !ok {
		return err("at: Index must be a number")
	}
	switch c := args[0].(type) {
	case *List:
		if idx < 0 || idx >= int64(len(c.Elements)) {
			return NILOBJ
		}
		return c.Elements[idx]
	case *String:
		if idx < 0 || idx >= int64(len(c.Value)) {
			return NILOBJ
		}
		return &String{Value: string(c.Value[idx])}
	}
	return err("at expects list or string")
}

func bSliceList(args ...Object) Object {
	if len(args) != 3 {
		return err("slice_list expects 3 arguments (list, start, end)")
	}
	container, ok := args[0].(*List)
	if !ok {
		return err("slice_list: erstes Argument must be a list")
	}
	start, ok := ToInt(args[1])
	if !ok {
		return err("slice_list: start must be a number")
	}
	end, ok := ToInt(args[2])
	if !ok {
		return err("slice_list: end must be a number")
	}
	total := int64(len(container.Elements))
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	if start >= end {
		return &List{Elements: []Object{}}
	}
	result := make([]Object, end-start)
	for i := start; i < end; i++ {
		result[i-start] = container.Elements[i]
	}
	return &List{Elements: result}
}

func bSort(args ...Object) Object {
	if len(args) != 1 {
		return err("sort expects 1 argument")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("sort expects list")
	}
	sorted := make([]Object, len(l.Elements))
	copy(sorted, l.Elements)
	allNumeric := true
	for _, e := range sorted {
		if _, ok := e.(*Integer); !ok {
			if _, ok := e.(*Float); !ok {
				allNumeric = false
				break
			}
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		if allNumeric {
			af, _ := ToFloat(sorted[i])
			bf, _ := ToFloat(sorted[j])
			return af < bf
		}
		return sorted[i].Inspect() < sorted[j].Inspect()
	})
	return &List{Elements: sorted}
}

func bMap(args ...Object) Object {
	if len(args) != 2 {
		return err("map expects 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("map: erstes Argument must be a list")
	}
	result := make([]Object, len(l.Elements))
	for i, e := range l.Elements {
		result[i] = callOne(args[1], e)
	}
	return &List{Elements: result}
}

func bFilter(args ...Object) Object {
	if len(args) != 2 {
		return err("filter expects 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("filter: erstes Argument must be a list")
	}
	var result []Object
	for _, e := range l.Elements {
		r := callOne(args[1], e)
		if IsTruthy(r) {
			result = append(result, e)
		}
	}
	return &List{Elements: result}
}

func bReduce(args ...Object) Object {
	if len(args) != 3 {
		return err("reduce expects 3 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("reduce: erstes Argument must be a list")
	}
	acc := args[2]
	for _, e := range l.Elements {
		acc = callTwo(args[1], acc, e)
	}
	return acc
}

func bEach(args ...Object) Object {
	if len(args) != 2 {
		return err("each expects 2 arguments")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("each: erstes Argument must be a list")
	}
	for _, e := range l.Elements {
		callOne(args[1], e)
	}
	return NILOBJ
}

func callOne(fn, arg Object) Object {
	if bi, ok := fn.(*BuiltinInfo); ok {
		return bi.Fn(arg)
	}
	if callUserFn != nil {
		return callUserFn(fn, arg)
	}
	return err("map/filter/each: function not callable (only builtins in VM mode)")
}

func callTwo(fn, a, b Object) Object {
	if bi, ok := fn.(*BuiltinInfo); ok {
		return bi.Fn(a, b)
	}
	if callUserFn != nil {
		return callUserFn(fn, a, b)
	}
	return err("reduce: function not callable")
}

var callUserFn func(fn Object, args ...Object) Object

func SetCallUserFn(f func(fn Object, args ...Object) Object) {
	callUserFn = f
}

func bRange(args ...Object) Object {
	if len(args) < 1 || len(args) > 3 {
		return err("range expects 1-3 arguments")
	}
	var start, end, step int64
	step = 1

	switch len(args) {
	case 1:
		n, ok := ToInt(args[0])
		if !ok {
			return err("range: Argument must be a number")
		}
		start = 0
		end = n
	case 2:
		s, ok1 := ToInt(args[0])
		e, ok2 := ToInt(args[1])
		if !ok1 || !ok2 {
			return err("range: Argumente must be numbers")
		}
		start = s
		end = e
	case 3:
		s, ok1 := ToInt(args[0])
		e, ok2 := ToInt(args[1])
		st, ok3 := ToInt(args[2])
		if !ok1 || !ok2 || !ok3 {
			return err("range: Argumente must be numbers")
		}
		start = s
		end = e
		step = st
	}

	var elems []Object
	for i := start; i < end; i += step {
		elems = append(elems, &Integer{Value: i})
	}
	return &List{Elements: elems}
}

// ---- Math ----

func bAbs(args ...Object) Object {
	if len(args) != 1 {
		return err("abs expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		if v.Value < 0 {
			return &Integer{Value: -v.Value}
		}
		return v
	case *Float:
		return &Float{Value: math.Abs(v.Value)}
	}
	return err("abs expects a number")
}

func bMin(args ...Object) Object {
	if len(args) < 2 {
		return err("min expects at least 2 arguments")
	}
	f, ok := ToFloat(args[0])
	if !ok {
		return err("min: Argumente must be numbers")
	}
	allInt := true
	for _, a := range args {
		if _, isI := a.(*Float); isI {
			allInt = false
		}
		af, ok := ToFloat(a)
		if !ok {
			return err("min: Argumente must be numbers")
		}
		if af < f {
			f = af
		}
	}
	if allInt {
		return &Integer{Value: int64(f)}
	}
	return &Float{Value: f}
}

func bMax(args ...Object) Object {
	if len(args) < 2 {
		return err("max expects at least 2 arguments")
	}
	f, ok := ToFloat(args[0])
	if !ok {
		return err("max: Argumente must be numbers")
	}
	allInt := true
	for _, a := range args {
		if _, isI := a.(*Float); isI {
			allInt = false
		}
		af, ok := ToFloat(a)
		if !ok {
			return err("max: Argumente must be numbers")
		}
		if af > f {
			f = af
		}
	}
	if allInt {
		return &Integer{Value: int64(f)}
	}
	return &Float{Value: f}
}

func bPow(args ...Object) Object {
	if len(args) != 2 {
		return err("pow expects 2 arguments")
	}
	b, ok1 := ToFloat(args[0])
	e, ok2 := ToFloat(args[1])
	if !ok1 || !ok2 {
		return err("pow: Argumente must be numbers")
	}
	return &Float{Value: math.Pow(b, e)}
}

func bSqrt(args ...Object) Object {
	if len(args) != 1 {
		return err("sqrt expects 1 argument")
	}
	v, ok := ToFloat(args[0])
	if !ok {
		return err("sqrt expects a number")
	}
	if v < 0 {
		return err("sqrt: negative number")
	}
	return &Float{Value: math.Sqrt(v)}
}

func bRound(args ...Object) Object {
	if len(args) != 1 {
		return err("round expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return v
	case *Float:
		return &Integer{Value: int64(math.Round(v.Value))}
	}
	return err("round expects a number")
}

// ---- Map ----

func bKeys(args ...Object) Object {
	if len(args) != 1 {
		return err("keys expects 1 argument")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("keys expects a map")
	}
	keys := make([]Object, 0, len(m.Pairs))
	for k := range m.Pairs {
		keys = append(keys, &String{Value: k})
	}
	return &List{Elements: keys}
}

func bValues(args ...Object) Object {
	if len(args) != 1 {
		return err("values expects 1 argument")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("values expects a map")
	}
	vals := make([]Object, 0, len(m.Pairs))
	for _, v := range m.Pairs {
		vals = append(vals, v)
	}
	return &List{Elements: vals}
}

func bGet(args ...Object) Object {
	if len(args) != 2 {
		return err("get expects 2 arguments (Map/List, Key/Index)")
	}
	switch container := args[0].(type) {
	case *Map:
		key, ok := args[1].(*String)
		if !ok {
			return err("get: Map-Key must be a string")
		}
		val, exists := container.Pairs[key.Value]
		if !exists {
			return NILOBJ
		}
		return val
	case *List:
		idx, ok := ToInt(args[1])
		if !ok {
			return err("get: Listen-Index must be a number")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return NILOBJ
		}
		return container.Elements[idx]
	}
	return err("get expects a map or list")
}

func bSet(args ...Object) Object {
	if len(args) != 3 {
		return err("set expects 3 arguments (Map, Key, Value)")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("set: first argument must be a map")
	}
	key, ok := args[1].(*String)
	if !ok {
		return err("set: Key must be a string")
	}
	m.Pairs[key.Value] = args[2]
	return m
}

// ---- Type checks ----

func bTypeOf(args ...Object) Object {
	if len(args) != 1 {
		return err("type_of expects 1 argument")
	}
	return &String{Value: string(args[0].Type())}
}

func bIsNum(args ...Object) Object {
	if len(args) != 1 {
		return err("is_num expects 1 argument")
	}
	_, i := args[0].(*Integer)
	_, f := args[0].(*Float)
	return NativeBoolToBoolean(i || f)
}

func bIsStr(args ...Object) Object {
	if len(args) != 1 {
		return err("is_str expects 1 argument")
	}
	_, ok := args[0].(*String)
	return NativeBoolToBoolean(ok)
}

func bIsList(args ...Object) Object {
	if len(args) != 1 {
		return err("is_list expects 1 argument")
	}
	_, ok := args[0].(*List)
	return NativeBoolToBoolean(ok)
}

func bIsMap(args ...Object) Object {
	if len(args) != 1 {
		return err("is_map expects 1 argument")
	}
	_, ok := args[0].(*Map)
	return NativeBoolToBoolean(ok)
}

func bIsNil(args ...Object) Object {
	if len(args) != 1 {
		return err("is_nil expects 1 argument")
	}
	return NativeBoolToBoolean(args[0] == NILOBJ)
}

// ---- Conversion ----

func bToStr(args ...Object) Object {
	if len(args) != 1 {
		return err("to_str expects 1 argument")
	}
	return &String{Value: args[0].Inspect()}
}

func bToNum(args ...Object) Object {
	if len(args) != 1 {
		return err("to_num expects 1 argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return v
	case *Float:
		return v
	case *String:
		f, e := strconv.ParseFloat(v.Value, 64)
		if e != nil {
			return err("to_num: '" + v.Value + "' is not a number")
		}
		if f == float64(int64(f)) && !strings.Contains(v.Value, ".") {
			return &Integer{Value: int64(f)}
		}
		return &Float{Value: f}
	case *Boolean:
		if v.Value {
			return &Integer{Value: 1}
		}
		return &Integer{Value: 0}
	}
	return err("to_num not possible")
}

// ---- Connection type for TCP ----

type ConnHandle int

var (
	connMu     sync.Mutex
	connNextID ConnHandle = 1
	connStore             = make(map[ConnHandle]net.Conn)
	listeners             = make(map[ConnHandle]net.Listener)
)

type TcpConn struct {
	Handle ConnHandle
}

func (tc *TcpConn) Type() ObjectType { return "TCP_CONN" }
func (tc *TcpConn) Inspect() string  { return fmt.Sprintf("tcp:%d", tc.Handle) }

type TcpListener struct {
	Handle ConnHandle
}

func (tl *TcpListener) Type() ObjectType { return "TCP_LISTENER" }
func (tl *TcpListener) Inspect() string  { return fmt.Sprintf("tcp-listener:%d", tl.Handle) }

// ---- Network ----

func bHttpGet(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowNet {
		return sandboxBlock("http_get (network)")
	}
	if len(args) != 1 {
		return err("http_get expects 1 argument (URL)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_get: URL must be a string")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, e := client.Get(url.Value)
	if e != nil {
		return err("http_get: " + e.Error())
	}
	defer resp.Body.Close()
	body, e := io.ReadAll(resp.Body)
	if e != nil {
		return err("http_get: " + e.Error())
	}
	result := make(map[string]Object)
	result["status"] = &Integer{Value: int64(resp.StatusCode)}
	result["body"] = &String{Value: string(body)}
	return &Map{Pairs: result}
}

func bHttpPost(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 || len(args) > 2 {
		return err("http_post expects 1-2 arguments (URL, Body?)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_post: URL must be a string")
	}
	var bodyStr string
	if len(args) >= 2 {
		if b, ok := args[1].(*String); ok {
			bodyStr = b.Value
		} else {
			bodyStr = args[1].Inspect()
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	var bodyReader io.Reader
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}
	resp, e := client.Post(url.Value, "application/json", bodyReader)
	if e != nil {
		return err("http_post: " + e.Error())
	}
	defer resp.Body.Close()
	respBody, e := io.ReadAll(resp.Body)
	if e != nil {
		return err("http_post: " + e.Error())
	}
	result := make(map[string]Object)
	result["status"] = &Integer{Value: int64(resp.StatusCode)}
	result["body"] = &String{Value: string(respBody)}
	return &Map{Pairs: result}
}

func bHttpGetJSON(args ...Object) Object {
	resp := bHttpGet(args...)
	if resp.Type() == ERROR {
		return resp
	}
	respMap := resp.(*Map)
	body := respMap.Pairs["body"].(*String)
	return jsonToObject(body.Value)
}

func bParseJSON(args ...Object) Object {
	if len(args) != 1 {
		return err("parse_json expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("parse_json expects a string")
	}
	return jsonToObject(s.Value)
}

func bToJSON(args ...Object) Object {
	if len(args) != 1 {
		return err("to_json expects 1 argument")
	}
	j, e := json.Marshal(objectToJSON(args[0]))
	if e != nil {
		return err("to_json: " + e.Error())
	}
	return &String{Value: string(j)}
}

func jsonToObject(raw string) Object {
	var data interface{}
	if e := json.Unmarshal([]byte(raw), &data); e != nil {
		return err("parse_json: " + e.Error())
	}
	return convertJSON(data)
}

func convertJSON(data interface{}) Object {
	switch v := data.(type) {
	case map[string]interface{}:
		pairs := make(map[string]Object)
		for k, val := range v {
			pairs[k] = convertJSON(val)
		}
		return &Map{Pairs: pairs}
	case []interface{}:
		elems := make([]Object, len(v))
		for i, val := range v {
			elems[i] = convertJSON(val)
		}
		return &List{Elements: elems}
	case float64:
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case bool:
		return NativeBoolToBoolean(v)
	case nil:
		return NILOBJ
	}
	return NILOBJ
}

func objectToJSON(obj Object) interface{} {
	switch v := obj.(type) {
	case *Map:
		m := make(map[string]interface{})
		for k, val := range v.Pairs {
			m[k] = objectToJSON(val)
		}
		return m
	case *List:
		l := make([]interface{}, len(v.Elements))
		for i, val := range v.Elements {
			l[i] = objectToJSON(val)
		}
		return l
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *String:
		return v.Value
	case *Boolean:
		return v.Value
	case *NilObject:
		return nil
	}
	return obj.Inspect()
}

// ---- TCP ----

func bTcpListen(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) != 2 {
		return err("tcp_listen expects 2 arguments (Host, Port)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_listen: Host must be a string")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_listen: Port must be a number")
	}
	addr := fmt.Sprintf("%s:%d", host.Value, port)
	ln, e := net.Listen("tcp", addr)
	if e != nil {
		return err("tcp_listen: " + e.Error())
	}
	connMu.Lock()
	h := connNextID
	connNextID++
	listeners[h] = ln
	connMu.Unlock()
	return &TcpListener{Handle: h}
}

func bTcpConnect(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) != 2 {
		return err("tcp_connect expects 2 arguments (Host, Port)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_connect: Host must be a string")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_connect: Port must be a number")
	}
	addr := net.JoinHostPort(host.Value, strconv.FormatInt(port, 10))
	c, e := net.Dial("tcp", addr)
	if e != nil {
		return err("tcp_connect: " + e.Error())
	}
	connMu.Lock()
	h := connNextID
	connNextID++
	connStore[h] = c
	connMu.Unlock()
	return &TcpConn{Handle: h}
}

func bTcpAccept(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_accept expects 1 argument (Listener)")
	}
	ln, ok := args[0].(*TcpListener)
	if !ok {
		return err("tcp_accept expects a TCP listener")
	}
	connMu.Lock()
	listener, exists := listeners[ln.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_accept: listener does not exist")
	}
	c, e := listener.Accept()
	if e != nil {
		return err("tcp_accept: " + e.Error())
	}
	connMu.Lock()
	h := connNextID
	connNextID++
	connStore[h] = c
	connMu.Unlock()
	return &TcpConn{Handle: h}
}

func bTcpRead(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_read expects 1 argument (connection)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_read expects a TCP connection")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_read: connection does not exist")
	}
	buf := make([]byte, 4096)
	n, e := c.Read(buf)
	if e != nil && e != io.EOF {
		return err("tcp_read: " + e.Error())
	}
	return &String{Value: string(buf[:n])}
}

func bTcpWrite(args ...Object) Object {
	if len(args) != 2 {
		return err("tcp_write expects 2 arguments (connection, data)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_write expects a TCP connection")
	}
	data, ok := args[1].(*String)
	if !ok {
		return err("tcp_write: data must be a string")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_write: connection does not exist")
	}
	_, e := c.Write([]byte(data.Value))
	if e != nil {
		return err("tcp_write: " + e.Error())
	}
	return NILOBJ
}

func bTcpClose(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_close expects 1 argument")
	}
	switch v := args[0].(type) {
	case *TcpConn:
		connMu.Lock()
		if c, ok := connStore[v.Handle]; ok {
			c.Close()
			delete(connStore, v.Handle)
		}
		connMu.Unlock()
	case *TcpListener:
		connMu.Lock()
		if ln, ok := listeners[v.Handle]; ok {
			ln.Close()
			delete(listeners, v.Handle)
		}
		connMu.Unlock()
	default:
		return err("tcp_close: invalid type")
	}
	return NILOBJ
}

// ---- Regex ----

func bRegexMatch(args ...Object) Object {
	if len(args) != 2 {
		return err("regex_match expects 2 arguments (Pattern, Text)")
	}
	pat, ok := args[0].(*String)
	txt, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("regex_match: Pattern und Text must be strings")
	}
	matched, e := regexp.MatchString(pat.Value, txt.Value)
	if e != nil {
		return err("regex_match: " + e.Error())
	}
	if matched {
		return TRUE
	}
	return FALSE
}

func bRegexReplace(args ...Object) Object {
	if len(args) != 3 {
		return err("regex_replace expects 3 arguments (Pattern, replacement, Text)")
	}
	pat, ok := args[0].(*String)
	repl, ok2 := args[1].(*String)
	txt, ok3 := args[2].(*String)
	if !ok || !ok2 || !ok3 {
		return err("regex_replace: Alle Argumente must be strings")
	}
	re, e := regexp.Compile(pat.Value)
	if e != nil {
		return err("regex_replace: " + e.Error())
	}
	return &String{Value: re.ReplaceAllString(txt.Value, repl.Value)}
}

// ---- Date/Time ----

func bNow(args ...Object) Object {
	_ = args
	ts := time.Now().Unix()
	return &Integer{Value: ts}
}

func bFormatTime(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("format_time expects 1-2 arguments (Timestamp, Format?)")
	}
	ts, ok := ToInt(args[0])
	if !ok {
		return err("format_time: Timestamp must be a number")
	}
	layout := "2006-01-02 15:04:05"
	if len(args) >= 2 {
		s, ok := args[1].(*String)
		if !ok {
			return err("format_time: Format must be a string")
		}
		layout = s.Value
	}
	t := time.Unix(ts, 0)
	return &String{Value: t.Format(layout)}
}

// ---- Random ----

func bRandom(args ...Object) Object {
	_ = args
	return &Float{Value: rand.Float64()}
}

func bRandomRange(args ...Object) Object {
	if len(args) != 2 {
		return err("random_range expects 2 arguments (Min, Max)")
	}
	min, ok1 := ToInt(args[0])
	max, ok2 := ToInt(args[1])
	if !ok1 || !ok2 {
		return err("random_range: Min und Max must be numbers")
	}
	if min >= max {
		return err("random_range: min must be less than max")
	}
	return &Integer{Value: min + rand.Int63n(max-min)}
}

// ---- Helpers ----

// ---- Result type ----

func bOk(args ...Object) Object {
	if len(args) != 1 {
		return err("Ok expects 1 argument")
	}
	return &Result{Ok: true, Val: args[0]}
}

func bErr(args ...Object) Object {
	if len(args) != 1 {
		return err("Err expects 1 argument (error message)")
	}
	msg, ok := args[0].(*String)
	if !ok {
		msg = &String{Value: args[0].Inspect()}
	}
	return &Result{Ok: false, Err: msg.Value}
}

func bIsOk(args ...Object) Object {
	if len(args) != 1 {
		return err("is_ok expects 1 argument")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return FALSE
	}
	return NativeBoolToBoolean(r.Ok)
}

func bIsErr(args ...Object) Object {
	if len(args) != 1 {
		return err("is_err expects 1 argument")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return TRUE
	}
	return NativeBoolToBoolean(!r.Ok)
}

func bUnwrap(args ...Object) Object {
	if len(args) != 1 {
		return err("unwrap expects 1 argument")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return err("unwrap expects a result")
	}
	if !r.Ok {
		return err("unwrap on Err: " + r.Err)
	}
	return r.Val
}

func bUnwrapOr(args ...Object) Object {
	if len(args) != 2 {
		return err("unwrap_or expects 2 arguments (Result, Default)")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return args[1]
	}
	if !r.Ok {
		return args[1]
	}
	return r.Val
}

// ---- Encoding ----

func bBase64Encode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64_encode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64_encode expects a string")
	}
	return &String{Value: base64.StdEncoding.EncodeToString([]byte(s.Value))}
}

func bBase64Decode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64_decode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64_decode expects a string")
	}
	b, e := base64.StdEncoding.DecodeString(s.Value)
	if e != nil {
		return err("base64_decode: " + e.Error())
	}
	return &String{Value: string(b)}
}

func strArg(args []Object, name string) (*String, bool) {
	if len(args) != 1 {
		return nil, false
	}
	s, ok := args[0].(*String)
	return s, ok
}

func err(msg string) *Error {
	return &Error{Message: msg}
}

func FormatMsg(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

// ---- AI Builtins ----

func bAiProvider(args ...Object) Object {
	if len(args) != 1 {
		return err("ai_provider expects 1 argument (name)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("ai_provider: argument must be a string")
	}
	ai.SetProvider(s.Value)
	return &String{Value: "provider set to " + s.Value}
}

func bAiModel(args ...Object) Object {
	if len(args) != 1 {
		return err("ai_model expects 1 argument (model name)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("ai_model: argument must be a string")
	}
	ai.SetModel(s.Value)
	return &String{Value: "model set to " + s.Value}
}

func bAiTimeout(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_timeout expects 1 argument (seconds)")
	}
	v, ok := ToInt(args[0])
	if !ok {
		return err("ai_timeout: argument must be a number")
	}
	ai.SetTimeout(int(v))
	return NILOBJ
}

func bAiHost(args ...Object) Object {
	if len(args) != 1 {
		return err("ai_host expects 1 argument (url)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("ai_host: argument must be a string (e.g. 'http://localhost:11434')")
	}
	ai.SetHost(s.Value)
	return &String{Value: "host set to " + s.Value}
}

func bAiCache(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_cache expects 1 argument (on|off|ttl in minutes)")
	}

	switch v := args[0].(type) {
	case *Boolean:
		ai.SetCacheEnabled(v.Value)
		if v.Value {
			return &String{Value: "ai cache enabled (ttl: 10 min)"}
		}
		return &String{Value: "ai cache disabled"}
	case *Integer:
		ttl := int(v.Value)
		if ttl <= 0 {
			ai.SetCacheEnabled(false)
			return &String{Value: "ai cache disabled"}
		}
		ai.SetCacheEnabled(true)
		ai.SetCacheTTL(ttl)
		return &String{Value: fmt.Sprintf("ai cache enabled (ttl: %d min)", ttl)}
	case *String:
		if v.Value == "clear" {
			ai.ClearCache()
			return &String{Value: "ai cache cleared"}
		}
		if v.Value == "stats" {
			h, m := ai.CacheStats()
			return &String{Value: fmt.Sprintf("cache hits: %d, misses: %d", h, m)}
		}
		if v.Value == "on" {
			ai.SetCacheEnabled(true)
			return &String{Value: "ai cache enabled (ttl: 10 min)"}
		}
		if v.Value == "off" {
			ai.SetCacheEnabled(false)
			return &String{Value: "ai cache disabled"}
		}
		return err("ai_cache: unknown option '" + v.Value + "'. Use 'on', 'off', 'clear', 'stats', or a number (minutes)")
	default:
		return err("ai_cache expects true/false, a number (minutes), or 'on'/'off'/'clear'/'stats'")
	}
}

func bWebSearch(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}

	if len(args) < 1 {
		return err("web_search expects 1 argument (query)")
	}

	query, ok := args[0].(*String)
	if !ok {
		return err("web_search: argument must be a string")
	}

	results, searchErr := ai.WebSearch(query.Value)
	if searchErr != nil {
		return err(searchErr.Error())
	}

	elems := make([]Object, len(results))
	for i, r := range results {
		elems[i] = &Map{Pairs: map[string]Object{
			"title":   &String{Value: r.Title},
			"snippet": &String{Value: r.Snippet},
			"url":     &String{Value: r.URL},
		}}
	}
	return &List{Elements: elems}
}

func bAiChat(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowAI {
		return sandboxBlock("ai_chat (AI calls)")
	}
	if len(args) < 2 {
		return err("ai_chat expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_chat: first argument must be a string (system prompt)")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_chat: second argument must be a string (user prompt)")
	}

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sp.Value},
			{Role: "user", Content: up.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("ai_chat: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bAiChatJSON(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("ai_chat_json expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_chat_json: first argument must be a string")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_chat_json: second argument must be a string")
	}

	sysPrompt := sp.Value + "\nYou must respond with valid JSON only. No markdown, no explanation."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: up.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("ai_chat_json: " + respErr.Error())
	}

	var parsed interface{}
	if jsonErr := json.Unmarshal([]byte(resp.Content), &parsed); jsonErr != nil {
		return err("ai_chat_json: invalid JSON response: " + resp.Content)
	}
	return convertJSON(parsed)
}

func bSummarize(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("summarize expects at least 1 argument (text)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("summarize: argument must be a string")
	}

	sysPrompt := "You are a precise summarizer. Summarize the given text concisely in 2-3 sentences. Respond only with the summary."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: "Summarize this:\n\n" + t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("summarize: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bTranslate(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("translate expects 2 arguments (text, target_language)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("translate: first argument must be a string (text)")
	}
	lang, ok := args[1].(*String)
	if !ok {
		return err("translate: second argument must be a string (target language)")
	}

	sysPrompt := "You are a translator. Translate the given text to " + lang.Value + ". Respond only with the translated text."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("translate: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bClassify(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("classify expects 2 arguments (text, categories)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("classify: first argument must be a string (text)")
	}

	var categories string
	switch a := args[1].(type) {
	case *String:
		categories = a.Value
	case *List:
		parts := make([]string, len(a.Elements))
		for i, e := range a.Elements {
			parts[i] = e.Inspect()
		}
		categories = strings.Join(parts, ", ")
	default:
		return err("classify: second argument must be a string or list of categories")
	}

	sysPrompt := "Classify the given text into EXACTLY ONE of the following categories. Respond with only the category name.\nCategories: " + categories

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("classify: " + respErr.Error())
	}
	return &String{Value: strings.TrimSpace(resp.Content)}
}

func bExtract(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("extract expects 2 arguments (text, schema)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("extract: first argument must be a string (text)")
	}
	schema, ok := args[1].(*String)
	if !ok {
		return err("extract: second argument must be a string (schema description)")
	}

	sysPrompt := "Extract the requested information from the text. Respond ONLY with valid JSON. No markdown, no explanation.\nSchema: " + schema.Value

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: t.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("extract: " + respErr.Error())
	}

	var parsed interface{}
	if jsonErr := json.Unmarshal([]byte(resp.Content), &parsed); jsonErr != nil {
		return err("extract: invalid JSON response: " + resp.Content)
	}
	return convertJSON(parsed)
}

func bGenerate(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("generate expects at least 1 argument (prompt)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("generate: argument must be a string")
	}

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "user", Content: p.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("generate: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bGenerateJSON(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("generate_json expects 2 arguments (instruction, schema)")
	}
	instruction, ok := args[0].(*String)
	if !ok {
		return err("generate_json: first argument must be a string (instruction)")
	}
	schema, ok := args[1].(*String)
	if !ok {
		return err("generate_json: second argument must be a string (schema)")
	}

	sysPrompt := "You are a JSON generator. Generate data matching the schema. Respond ONLY with valid JSON. No markdown, no explanation, no extra text.\nSchema: " + schema.Value

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: instruction.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("generate_json: " + respErr.Error())
	}

	var parsed interface{}
	if jsonErr := json.Unmarshal([]byte(resp.Content), &parsed); jsonErr != nil {
		return err("generate_json: invalid JSON response: " + resp.Content)
	}
	return convertJSON(parsed)
}

func bAsk(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("ask expects at least 1 argument (question)")
	}
	q, ok := args[0].(*String)
	if !ok {
		return err("ask: argument must be a string")
	}

	sysPrompt := "You are a helpful assistant. Answer the question concisely and accurately."

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: q.Value},
		},
	}

	resp, respErr := ai.Chat(req)
	if respErr != nil {
		return err("ask: " + respErr.Error())
	}
	return &String{Value: resp.Content}
}

func bAiStream(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("ai_stream expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_stream: first argument must be a string (system prompt)")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_stream: second argument must be a string (user prompt)")
	}

	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sp.Value},
			{Role: "user", Content: up.Value},
		},
	}

	var fullText strings.Builder
	streamErr := ai.Stream(req, func(token string) error {
		fmt.Print(token)
		fullText.WriteString(token)
		return nil
	})
	fmt.Println()

	if streamErr != nil {
		return err("ai_stream: " + streamErr.Error())
	}
	return &String{Value: fullText.String()}
}

func bAiRateLimit(args ...Object) Object {
	if len(args) < 1 {
		return err("ai_rate_limit expects 1 argument (calls_per_second)")
	}
	v, ok := ToInt(args[0])
	if !ok {
		return err("ai_rate_limit: argument must be a number")
	}
	ai.SetRateLimit(int(v))
	return NILOBJ
}

func bAiParallel(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 3 {
		return err("ai_parallel expects 3 arguments (concurrency, system_prompt, items)")
	}

	concurrency, ok := ToInt(args[0])
	if !ok {
		return err("ai_parallel: first argument must be a number (concurrency)")
	}

	sp, ok := args[1].(*String)
	if !ok {
		return err("ai_parallel: second argument must be a string (system prompt)")
	}

	items, ok := args[2].(*List)
	if !ok {
		return err("ai_parallel: third argument must be a list of strings")
	}

	requests := make([]ai.ChatRequest, len(items.Elements))
	for i, elem := range items.Elements {
		s, ok := elem.(*String)
		if !ok {
			s = &String{Value: elem.Inspect()}
		}
		requests[i] = ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: sp.Value},
				{Role: "user", Content: s.Value},
			},
		}
	}

	results, errs := ai.ChatParallel(requests, int(concurrency))

	elems := make([]Object, len(results))
	for i := range results {
		if errs[i] != nil {
			elems[i] = err("ai_parallel[" + fmt.Sprintf("%d", i) + "]: " + errs[i].Error())
		} else {
			elems[i] = &String{Value: results[i].Content}
		}
	}
	return &List{Elements: elems}
}

func bAiBatch(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 2 {
		return err("ai_batch expects 2 arguments (system_prompt, items)")
	}

	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_batch: first argument must be a string (system prompt)")
	}

	items, ok := args[1].(*List)
	if !ok {
		return err("ai_batch: second argument must be a list of strings")
	}

	requests := make([]ai.ChatRequest, len(items.Elements))
	for i, elem := range items.Elements {
		s, ok := elem.(*String)
		if !ok {
			s = &String{Value: elem.Inspect()}
		}
		requests[i] = ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: sp.Value},
				{Role: "user", Content: s.Value},
			},
		}
	}

	results, errs := ai.ChatParallel(requests, 0)

	elems := make([]Object, len(results))
	for i := range results {
		if errs[i] != nil {
			elems[i] = err("ai_batch[" + fmt.Sprintf("%d", i) + "]: " + errs[i].Error())
		} else {
			elems[i] = &String{Value: results[i].Content}
		}
	}
	return &List{Elements: elems}
}

func bEmbed(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 {
		return err("embed expects 1 argument (text)")
	}
	t, ok := args[0].(*String)
	if !ok {
		return err("embed: argument must be a string")
	}

	vec, vecErr := ai.Embed(t.Value)
	if vecErr != nil {
		return err("embed: " + vecErr.Error())
	}

	elems := make([]Object, len(vec))
	for i, v := range vec {
		elems[i] = &Float{Value: v}
	}
	return &List{Elements: elems}
}

func bEmbedBatch(args ...Object) Object {
	if len(args) < 1 {
		return err("embed_batch expects 1 argument (list of texts)")
	}
	items, ok := args[0].(*List)
	if !ok {
		return err("embed_batch: argument must be a list of strings")
	}

	texts := make([]string, len(items.Elements))
	for i, elem := range items.Elements {
		s, okElem := elem.(*String)
		if !okElem {
			s = &String{Value: elem.Inspect()}
		}
		texts[i] = s.Value
	}

	vectors, errs := ai.EmbedBatch(texts, 4)

	elems := make([]Object, len(vectors))
	for i := range vectors {
		if errs[i] != nil {
			elems[i] = err("embed_batch[" + fmt.Sprintf("%d", i) + "]: " + errs[i].Error())
		} else {
			vecElems := make([]Object, len(vectors[i]))
			for j, v := range vectors[i] {
				vecElems[j] = &Float{Value: v}
			}
			elems[i] = &List{Elements: vecElems}
		}
	}
	return &List{Elements: elems}
}

func bCosineSim(args ...Object) Object {
	if len(args) < 2 {
		return err("cosine_sim expects 2 arguments (vector_a, vector_b)")
	}
	vecA, okA := args[0].(*List)
	vecB, okB := args[1].(*List)
	if !okA || !okB {
		return err("cosine_sim: arguments must be lists of numbers")
	}

	a := listToFloats(vecA)
	b := listToFloats(vecB)

	return &Float{Value: ai.CosineSimilarity(a, b)}
}

func bDotProduct(args ...Object) Object {
	if len(args) < 2 {
		return err("dot_product expects 2 arguments (vector_a, vector_b)")
	}
	vecA, okA := args[0].(*List)
	vecB, okB := args[1].(*List)
	if !okA || !okB {
		return err("dot_product: arguments must be lists of numbers")
	}

	a := listToFloats(vecA)
	b := listToFloats(vecB)

	return &Float{Value: ai.DotProduct(a, b)}
}

func bNearest(args ...Object) Object {
	if len(args) < 3 {
		return err("nearest expects 3 arguments (query_vec, doc_vecs, k)")
	}
	query, okQ := args[0].(*List)
	docs, okD := args[1].(*List)
	if !okQ || !okD {
		return err("nearest: first two arguments must be lists")
	}
	k, okK := ToInt(args[2])
	if !okK {
		return err("nearest: third argument must be a number (k)")
	}

	q := listToFloats(query)

	docVectors := make([][]float64, len(docs.Elements))
	for i, elem := range docs.Elements {
		docList, ok := elem.(*List)
		if !ok {
			return err("nearest: document vectors must be lists of numbers")
		}
		docVectors[i] = listToFloats(docList)
	}

	indices := ai.Nearest(q, docVectors, int(k))

	elems := make([]Object, len(indices))
	for i, idx := range indices {
		elems[i] = &Integer{Value: int64(idx)}
	}
	return &List{Elements: elems}
}

func listToFloats(list *List) []float64 {
	floats := make([]float64, len(list.Elements))
	for i, elem := range list.Elements {
		if f, ok := elem.(*Float); ok {
			floats[i] = f.Value
		} else if n, ok := elem.(*Integer); ok {
			floats[i] = float64(n.Value)
		}
	}
	return floats
}

// ---- Tool Registry ----

type ToolEntry struct {
	Def ai.ToolDef
	Fn  Object
}

var toolRegistry = map[string]ToolEntry{}

func bAiTool(args ...Object) Object {
	if len(args) < 4 {
		return err("ai_tool expects 4 arguments (name, description, parameters, function)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("ai_tool: first argument must be a string (tool name)")
	}
	desc, ok := args[1].(*String)
	if !ok {
		return err("ai_tool: second argument must be a string (description)")
	}
	params, ok := args[2].(*Map)
	if !ok {
		return err("ai_tool: third argument must be a map (parameter schema)")
	}

	fn := args[3]

	paramSchema := make(map[string]interface{})
	for k, v := range params.Pairs {
		if s, ok := v.(*String); ok {
			paramSchema[k] = map[string]interface{}{
				"type":        "string",
				"description": s.Value,
			}
		} else if m, ok := v.(*Map); ok {
			inner := make(map[string]interface{})
			for ik, iv := range m.Pairs {
				if is, ok := iv.(*String); ok {
					inner[ik] = is.Value
				}
			}
			paramSchema[k] = inner
		}
	}

	toolRegistry[name.Value] = ToolEntry{
		Def: ai.ToolDef{
			Name:        name.Value,
			Description: desc.Value,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": paramSchema,
				"required":   keysToStrings(params),
			},
		},
		Fn: fn,
	}

	return NILOBJ
}

func keysToStrings(m *Map) []string {
	keys := make([]string, 0, len(m.Pairs))
	for k := range m.Pairs {
		keys = append(keys, k)
	}
	return keys
}

func bAiWithTools(args ...Object) Object {
	if len(args) < 2 {
		return err("ai_with_tools expects at least 2 arguments (system_prompt, user_prompt)")
	}
	sp, ok := args[0].(*String)
	if !ok {
		return err("ai_with_tools: first argument must be a string (system prompt)")
	}
	up, ok := args[1].(*String)
	if !ok {
		return err("ai_with_tools: second argument must be a string (user prompt)")
	}

	maxRounds := 5
	argIdx := 2
	if len(args) >= 3 {
		if n, ok := ToInt(args[2]); ok {
			maxRounds = int(n)
			argIdx = 3
		}
	}

	// Optional sandbox name from a string arg
	profile := ActiveProfile
	if len(args) > argIdx {
		if s, ok := args[argIdx].(*String); ok {
			if p, pErr := GetProfile(s.Value); pErr == nil {
				profile = p
			}
		}
	}

	if profile.Name != "none" {
		if canErr := profile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
		if canErr := profile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}

	tools := make([]ai.ToolDef, 0, len(toolRegistry))
	for _, entry := range toolRegistry {
		tools = append(tools, entry.Def)
	}

	if len(tools) == 0 {
		return err("ai_with_tools: no tools registered. Use ai_tool first.")
	}

	executor := func(toolName string, args map[string]interface{}) (string, error) {
		entry, exists := toolRegistry[toolName]
		if !exists {
			return "", fmt.Errorf("unknown tool: %s", toolName)
		}

		if profile.Name != "none" {
			if canErr := profile.CanExec(); canErr != nil {
				return "", fmt.Errorf("E_SANDBOX: tool '%s' execution blocked by profile '%s'", toolName, profile.Name)
			}
		}

		argObjects := make([]Object, 0, len(args))
		for _, v := range args {
			switch val := v.(type) {
			case string:
				argObjects = append(argObjects, &String{Value: val})
			case float64:
				if val == float64(int64(val)) {
					argObjects = append(argObjects, &Integer{Value: int64(val)})
				} else {
					argObjects = append(argObjects, &Float{Value: val})
				}
			case bool:
				argObjects = append(argObjects, NativeBoolToBoolean(val))
			default:
				argObjects = append(argObjects, &String{Value: fmt.Sprintf("%v", val)})
			}
		}

		if callUserFn != nil {
			result := callUserFn(entry.Fn, argObjects...)
			return result.Inspect(), nil
		}

		if bi, ok := entry.Fn.(*BuiltinInfo); ok {
			result := bi.Fn(argObjects...)
			return result.Inspect(), nil
		}

		return "", fmt.Errorf("tool function not callable")
	}

	result, chatErr := ai.ChatWithTools(sp.Value, up.Value, tools, executor, maxRounds)
	if chatErr != nil {
		return err("ai_with_tools: " + chatErr.Error())
	}

	return &String{Value: result}
}

// ---- AI — Agents ----

func bAgent(args ...Object) Object {
	if len(args) < 2 {
		return err("agent expects 2 arguments (name, system_prompt)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("agent: first argument must be a string (name)")
	}
	prompt, ok := args[1].(*String)
	if !ok {
		return err("agent: second argument must be a string (system prompt)")
	}

	ai.CreateAgent(name.Value, prompt.Value)
	return &String{Value: "agent '" + name.Value + "' created"}
}

func bAgentAsk(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	}

	if len(args) < 2 {
		return err("agent_ask expects 2 arguments (name, message)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("agent_ask: first argument must be a string (agent name)")
	}
	msg, ok := args[1].(*String)
	if !ok {
		return err("agent_ask: second argument must be a string (message)")
	}

	ag, exists := ai.GetAgent(name.Value)
	if !exists {
		return err("agent_ask: agent '" + name.Value + "' not found. Create it with agent first.")
	}

	resp, askErr := ag.Ask(msg.Value)
	if askErr != nil {
		return err("agent_ask: " + askErr.Error())
	}

	return &String{Value: resp}
}

func bAgentClear(args ...Object) Object {
	if len(args) < 1 {
		return err("agent_clear expects 1 argument (name)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("agent_clear: argument must be a string (agent name)")
	}

	ag, exists := ai.GetAgent(name.Value)
	if !exists {
		return err("agent_clear: agent '" + name.Value + "' not found")
	}

	ag.Clear()
	return &String{Value: "agent '" + name.Value + "' history cleared"}
}

// ---- Test Assertions ----

func bAssert(args ...Object) Object {
	if len(args) != 1 {
		return err("assert expects 1 argument (condition)")
	}
	if !IsTruthy(args[0]) {
		return err("assertion failed: value is not truthy")
	}
	return NILOBJ
}

func bAssertEq(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_eq expects 2 arguments (expected, actual)")
	}
	if !ValuesEqual(args[0], args[1]) {
		return err(fmt.Sprintf("assertion failed: expected %s, got %s", args[0].Inspect(), args[1].Inspect()))
	}
	return NILOBJ
}

func bAssertNotEq(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_not_eq expects 2 arguments (unexpected, actual)")
	}
	if ValuesEqual(args[0], args[1]) {
		return err(fmt.Sprintf("assertion failed: got %s, but expected different value", args[0].Inspect()))
	}
	return NILOBJ
}

func bAssertLt(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_lt expects 2 arguments (a, b)")
	}
	a := toFloat(args[0])
	b := toFloat(args[1])
	if a >= b {
		return err(fmt.Sprintf("assertion failed: expected %s < %s", args[0].Inspect(), args[1].Inspect()))
	}
	return NILOBJ
}

func bAssertGt(args ...Object) Object {
	if len(args) != 2 {
		return err("assert_gt expects 2 arguments (a, b)")
	}
	a := toFloat(args[0])
	b := toFloat(args[1])
	if a <= b {
		return err(fmt.Sprintf("assertion failed: expected %s > %s", args[0].Inspect(), args[1].Inspect()))
	}
	return NILOBJ
}

func bAssertError(args ...Object) Object {
	if len(args) != 1 {
		return err("assert_error expects 1 argument (function)")
	}
	fn, ok := args[0].(*Function)
	if !ok {
		return err("assert_error expects a function (use { ... })")
	}
	result := callUserFn(fn)
	if result == nil {
		return err("assertion failed: expected an error, but got nil")
	}
	if result.Type() != ERROR {
		return err(fmt.Sprintf("assertion failed: expected an error, but got %s", result.Inspect()))
	}
	return NILOBJ
}

func toFloat(o Object) float64 {
	switch v := o.(type) {
	case *Integer:
		return float64(v.Value)
	case *Float:
		return v.Value
	default:
		return 0
	}
}

// ---- Sandbox ----

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
