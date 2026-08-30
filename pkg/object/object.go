package object

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

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
	BYTES                        = "BYTES"
	FUTURE                       = "FUTURE"
	ERROR                        = "ERROR"
	STRUCT                       = "STRUCT"
	CHANNEL                      = "CHANNEL"
	MUTEX                        = "MUTEX"
	SEMAPHORE                    = "SEMAPHORE"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

// MaxCallDepth caps active function call depth in both the tree-walker
// (pkg/eval) and the bytecode VM (pkg/vm). Without it, unbounded recursion
// exhausts the Go stack and crashes the process.
const MaxCallDepth = 1024

type Integer struct{ Value int64 }

func (i *Integer) Type() ObjectType { return INTEGER }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Float struct{ Value float64 }

func (f *Float) Type() ObjectType { return FLOAT }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

type String struct{ Value string }

func (s *String) Type() ObjectType { return STRING }
func (s *String) Inspect() string  { return s.Value }

type Bytes struct{ Value []byte }

func (b *Bytes) Type() ObjectType { return BYTES }
func (b *Bytes) Inspect() string  { return hex.EncodeToString(b.Value) }

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

// MapEntry is a single key/value pair in a runtime map. Maps preserve the
// insertion order of their keys: source map literals keep declaration order,
// while programmatically constructed maps (HTTP headers, MCP tool args, ...)
// use MapFromGo, which sorts keys deterministically.
type MapEntry struct {
	Key   string
	Value Object
}

type Map struct {
	Pairs []MapEntry
	index map[string]int
}

func (m *Map) Type() ObjectType { return MAP }

// rebuildIndex lazily derives the key->index lookup table.
func (m *Map) rebuildIndex() {
	m.index = make(map[string]int, len(m.Pairs))
	for i, p := range m.Pairs {
		m.index[p.Key] = i
	}
}

// Get returns the value for key and whether it was present. Lookups without a
// successful preceding index build fall back to a linear scan.
func (m *Map) Get(key string) (Object, bool) {
	if m.index != nil {
		if i, ok := m.index[key]; ok && i < len(m.Pairs) {
			return m.Pairs[i].Value, true
		}
		return nil, false
	}
	for _, p := range m.Pairs {
		if p.Key == key {
			return p.Value, true
		}
	}
	return nil, false
}

// Set inserts or replaces key, preserving the first-insertion position
// ("last wins" on value, position kept). It appends a new key at the end.
func (m *Map) Set(key string, value Object) *Map {
	for i := range m.Pairs {
		if m.Pairs[i].Key == key {
			m.Pairs[i].Value = value
			m.rebuildIndex()
			return m
		}
	}
	m.Pairs = append(m.Pairs, MapEntry{Key: key, Value: value})
	m.rebuildIndex()
	return m
}

// Del removes key if present.
func (m *Map) Del(key string) bool {
	for i := range m.Pairs {
		if m.Pairs[i].Key == key {
			m.Pairs = append(m.Pairs[:i], m.Pairs[i+1:]...)
			m.rebuildIndex()
			return true
		}
	}
	return false
}

// Keys returns the keys in map order.
func (m *Map) Keys() []string {
	keys := make([]string, 0, len(m.Pairs))
	for _, p := range m.Pairs {
		keys = append(keys, p.Key)
	}
	return keys
}

// Values returns the values in map order.
func (m *Map) Values() []Object {
	vals := make([]Object, 0, len(m.Pairs))
	for _, p := range m.Pairs {
		vals = append(vals, p.Value)
	}
	return vals
}

// NewMap returns an empty ordered map.
func NewMap() *Map {
	return &Map{Pairs: []MapEntry{}, index: map[string]int{}}
}

// MapFromGo builds an ordered map from a Go map, sorting the keys
// deterministically. Use for programmatically constructed maps.
func MapFromGo(m map[string]Object) *Map {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := &Map{Pairs: make([]MapEntry, 0, len(m)), index: make(map[string]int, len(m))}
	for i, k := range keys {
		out.Pairs = append(out.Pairs, MapEntry{Key: k, Value: m[k]})
		out.index[k] = i
	}
	return out
}

func (m *Map) Inspect() string {
	pairs := make([]string, 0, len(m.Pairs))
	for _, p := range m.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s", p.Key, p.Value.Inspect()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

// Error is a catchable runtime error. Message carries the human-readable
// text (and historically the E-code prefix); Code/File/Line/Col are
// structured fields so the CLI can render source snippets.
type Error struct {
	Message string
	Code    string
	File    string
	Line    int
	Col     int
}

func (e *Error) Type() ObjectType { return ERROR }
func (e *Error) Inspect() string  { return e.Message }

// Error satisfies the standard error interface so a runtime *Error can be
// returned directly from engine entry points.
func (e *Error) Error() string { return e.Message }

// HasPosition reports whether the error carries a resolvable source location.
func (e *Error) HasPosition() bool { return e.Line > 0 }

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

// Channel is an in-memory communication primitive shared by reference between
// concurrent tasks (see spawn/>>). send blocks, recv blocks, try_recv/try_send
// do not.
type Channel struct {
	ch     chan Object
	closed bool
	mu     sync.Mutex
}

func NewChannel(capacity int) *Channel {
	if capacity < 0 {
		capacity = 0
	}
	return &Channel{ch: make(chan Object, capacity)}
}

func (c *Channel) Type() ObjectType { return CHANNEL }
func (c *Channel) Inspect() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return "chan(closed)"
	}
	return fmt.Sprintf("chan(%d/%d)", len(c.ch), cap(c.ch))
}

func (c *Channel) TryRecv() Object {
	select {
	case v, open := <-c.ch:
		if !open {
			return NILOBJ
		}
		return v
	default:
		return NILOBJ
	}
}

// Mutex is a shared in-memory mutual exclusion primitive.
type Mutex struct {
	mu sync.Mutex
}

func NewMutex() *Mutex { return &Mutex{} }

func (m *Mutex) Type() ObjectType { return MUTEX }
func (m *Mutex) Inspect() string  { return "mutex" }

// Semaphore is a counting semaphore (bounded parallelism primitive).
type Semaphore struct {
	ch chan struct{}
	n  int
}

func NewSemaphore(n int) *Semaphore {
	if n < 1 {
		n = 1
	}
	return &Semaphore{ch: make(chan struct{}, n), n: n}
}

func (s *Semaphore) Type() ObjectType { return SEMAPHORE }
func (s *Semaphore) Inspect() string  { return fmt.Sprintf("semaphore(%d/%d)", len(s.ch), s.n) }

type BuiltinInfo struct {
	Name string
	Fn   func(args ...Object) Object
}

func (bi *BuiltinInfo) Type() ObjectType { return "BUILTIN" }
func (bi *BuiltinInfo) Inspect() string  { return "builtin: " + bi.Name }

// CallableBuiltin lets a builtin-function value defined outside this package
// (namely the tree-walker's own eval.Builtin wrapper, returned when a bare
// builtin identifier like read_file is evaluated as a value rather than
// called) be dispatched by CallUserFunction the same way *BuiltinInfo is.
// pkg/object cannot import pkg/eval (pkg/eval already imports pkg/object),
// so this interface is the seam: eval.Builtin implements it, and
// CallUserFunction dispatches through it without knowing the concrete type.
// See the round-9 audit finding this fixes — ai_tool registered directly
// with a builtin (ai_tool "read_file" ... read_file, the pattern every
// redteam*.pipe script and several examples use) silently failed every call
// with "not callable: BUILTIN" under the tree-walker, because
// CallUserFunction's type switch only recognized *BuiltinInfo.
type CallableBuiltin interface {
	BuiltinFn() func(args ...Object) Object
}

type CompiledFunction struct {
	Instructions interface{}
	Lines        []int // source line per instruction byte; may be nil
	NumLocals    int
	NumFree      int
}

// UserFunctionExecutor invokes user-defined functions on the engine that
// created them (tree-walker or VM). Each runtime implements this so that
// builtins like map/filter/reduce can call back into the correct engine
// without a process-wide global hook (which would race across parallel VMs).
type UserFunctionExecutor interface {
	CallUserFunction(fn Object, args ...Object) Object
}

// UserFunctionSpawner launches a user-defined function in the background and
// returns a Future for its eventual result. Used by `go` (which discards the
// Future) and `spawn` (which returns it) when handed a VM closure.
type UserFunctionSpawner interface {
	SpawnUserFunction(fn Object, args ...Object) *Future
}

// AwaitBuiltinName identifies the `await` builtin. Its Future argument must
// reach the builtin unresolved so a timeout can be applied; every other
// callable auto-resolves Future arguments before dispatch.
const AwaitBuiltinName = "await"

// IsAwaitBuiltin reports whether fn is the await builtin.
func IsAwaitBuiltin(fn Object) bool {
	if bi, ok := fn.(*BuiltinInfo); ok {
		return bi.Name == AwaitBuiltinName
	}
	return false
}

type Closure struct {
	Fn       *CompiledFunction
	Free     []Object
	Executor UserFunctionExecutor
}

// CallUserFunction dispatches a user-defined function to whichever runtime
// created it: the tree-walker (*Function) or the VM (*Closure).
func CallUserFunction(fn Object, args ...Object) Object {
	switch f := fn.(type) {
	case *Function:
		if ex, ok := f.EvalCtx.(UserFunctionExecutor); ok {
			return ex.CallUserFunction(f, args...)
		}
		return err("function execution not available")
	case *Closure:
		if f.Executor != nil {
			return f.Executor.CallUserFunction(f, args...)
		}
		return err("function execution not available")
	case *BuiltinInfo:
		return f.Fn(args...)
	case CallableBuiltin:
		return f.BuiltinFn()(args...)
	}
	return err(fmt.Sprintf("not callable: %s", fn.Type()))
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION }
func (cf *CompiledFunction) Inspect() string  { return "compiled function" }
func (c *Closure) Type() ObjectType           { return CLOSURE }
func (c *Closure) Inspect() string            { return "closure" }

type StructDef struct {
	Name     string
	Fields   []string
	Defaults map[string]Object
}

func (sd *StructDef) Type() ObjectType { return STRUCT }
func (sd *StructDef) Inspect() string {
	fields := []string{}
	for _, f := range sd.Fields {
		if def, ok := sd.Defaults[f]; ok {
			fields = append(fields, fmt.Sprintf("%s: %s", f, def.Inspect()))
		} else {
			fields = append(fields, f)
		}
	}
	return fmt.Sprintf("struct %s {%s}", sd.Name, strings.Join(fields, ", "))
}

type StructInstance struct {
	Def    *StructDef
	Values map[string]Object
}

func (si *StructInstance) Type() ObjectType { return STRUCT }
func (si *StructInstance) Inspect() string {
	fields := []string{}
	for _, f := range si.Def.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", f, si.Values[f].Inspect()))
	}
	return fmt.Sprintf("%s{%s}", si.Def.Name, strings.Join(fields, ", "))
}

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
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type() != b.Type() {
		return false
	}
	switch av := a.(type) {
	case *Integer:
		return av.Value == b.(*Integer).Value
	case *Float:
		return av.Value == b.(*Float).Value
	case *String:
		return av.Value == b.(*String).Value
	case *Bytes:
		return bytes.Equal(av.Value, b.(*Bytes).Value)
	case *Boolean:
		return av.Value == b.(*Boolean).Value
	case *NilObject:
		return true
	case *List:
		bv := b.(*List)
		if len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i, e := range av.Elements {
			if !ValuesEqual(e, bv.Elements[i]) {
				return false
			}
		}
		return true
	case *Map:
		bv := b.(*Map)
		if len(av.Pairs) != len(bv.Pairs) {
			return false
		}
		for _, p := range av.Pairs {
			bVal, ok := bv.Get(p.Key)
			if !ok || !ValuesEqual(p.Value, bVal) {
				return false
			}
		}
		return true
	}
	return false
}

// ---- Builtins ----

var Builtins = []BuiltinInfo{
	// IO / File System
	{"print", bPrint},
	{"print_raw", bPrintRaw},
	{"input", bInput},
	{"exec", bExec},
	{"proc_start", bProcStart},
	{"proc_wait", bProcWait},
	{"proc_kill", bProcKill},
	{"proc_running", bProcRunning},
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
	{"dotenv", bDotenv},
	{"list_dir", bListDir},
	{"make_dir", bMakeDir},
	{"remove_dir", bRemoveDir},
	{"path_join", bPathJoin},
	{"path_base", bPathBase},
	{"path_dir", bPathDir},
	{"path_ext", bPathExt},
	{"env", bEnv},
	{"sleep", bSleep},
	{"args", bArgs},
	{"read_stdin", bReadStdin},

	// Network
	{"http_get", bHttpGet},
	{"http_post", bHttpPost},
	{"http_get_json", bHttpGetJSON},
	{"http_request", bHttpRequest},
	{"parse_json", bParseJSON},
	{"to_json", bToJSON},

	// TCP
	{"tcp_listen", bTcpListen},
	{"tcp_connect", bTcpConnect},
	{"tcp_connect_tls", bTcpConnectTLS},
	{"tcp_accept", bTcpAccept},
	{"tcp_read", bTcpRead},
	{"tcp_read_bytes", bTcpReadBytes},
	{"tcp_set_read_timeout", bTcpSetReadTimeout},
	{"tcp_write", bTcpWrite},
	{"tcp_close", bTcpClose},

	// HTTP Server
	{"http_server", bHttpServer},
	{"http_close", bHttpClose},

	// String
	{"upper", bUpper},
	{"lower", bLower},
	{"trim", bTrim},
	{"split", bSplit},
	{"join", bJoin},
	{"contains", bContains},
	{"repeat", bRepeat},
	{"replace", bReplace},
	{"replace_all", bReplaceAll},

	// CSV
	{"csv_parse", bCsvParse},
	{"csv_format", bCsvFormat},

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
	{"unique", bUnique},
	{"go", bGo},
	{"spawn", bSpawn},
	{"await", bAwait},

	// Concurrency — channels, mutex, semaphore
	{"chan", bChan},
	{"send", bSend},
	{"recv", bRecv},
	{"try_recv", bTryRecv},
	{"try_send", bTrySend},
	{"close", bClose},
	{"chan_len", bChanLen},
	{"chan_cap", bChanCap},
	{"mutex", bMutex},
	{"lock", bLock},
	{"unlock", bUnlock},
	{"try_lock", bTryLock},
	{"semaphore", bSemaphore},
	{"acquire", bAcquire},
	{"release", bRelease},
	{"try_acquire", bTryAcquire},

	// Math
	{"abs", bAbs},
	{"min", bMin},
	{"max", bMax},
	{"pow", bPow},
	{"sqrt", bSqrt},
	{"round", bRound},
	{"ceil", bCeil},
	{"floor", bFloor},

	// Map
	{"keys", bKeys},
	{"values", bValues},
	{"get", bGet},
	{"set", bSet},
	{"map_delete", bMapDelete},

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
	{"base64url_encode", bBase64URLEncode},
	{"base64url_decode", bBase64URLDecode},
	{"url_encode", bURLEncode},
	{"url_decode", bURLDecode},

	// Hashing
	{"sha256", bSha256},
	{"md5", bMd5},
	{"sha1", bSha1},
	{"sha512", bSha512},

	// Regex
	{"regex_match", bRegexMatch},
	{"regex_captures", bRegexCaptures},
	{"regex_replace", bRegexReplace},

	// Date/Time
	{"now", bNow},
	{"time_ms", bTimeMs},
	{"format_time", bFormatTime},
	{"parse_date", bParseDate},

	// Random
	{"random", bRandom},
	{"random_range", bRandomRange},

	// AI — Configuration
	{"ai_provider", bAiProvider},
	{"ai_model", bAiModel},
	{"ai_timeout", bAiTimeout},
	{"ai_host", bAiHost},
	{"ai_cache", bAiCache},
	{"ai_set_key", bAiSetKey},

	// AI — Cost & Observability
	{"ai_cost", bAiCost},
	{"ai_tokens", bAiTokens},
	{"ai_cache_hits", bAiCacheHits},
	{"ai_cache_misses", bAiCacheMisses},

	// AI — Retrieval
	{"web_search", bWebSearch},
	{"wiki_search", bWikiSearch},

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
	{"sandbox_lock", bSandboxLock},
	{"audit_log", bAuditLog},
	{"budget_spent", bBudgetSpent},

	// AI — Tool Calling
	{"ai_tool", bAiTool},
	{"ai_with_tools", bAiWithTools},

	// MCP — Model Context Protocol (Server + Client)
	{"mcp_server", bMcpServer},
	{"mcp_serve_stdio", bMcpServeStdio},
	{"mcp_serve_sse", bMcpServeSSE},
	{"mcp_tools", bMcpTools},
	{"mcp_resources", bMcpResources},
	{"mcp_read_resource", bMcpReadResource},
	{"mcp_prompts", bMcpPrompts},
	{"mcp_prompt_get", bMcpPromptGet},
	{"mcp_resource", bMcpResource},
	{"mcp_resource_template", bMcpResourceTemplate},
	{"mcp_prompt", bMcpPrompt},
	{"mcp_use_stdio", bMcpUseStdio},
	{"mcp_use_sse", bMcpUseSSE},

	// AI — Agents
	{"agent", bAgent},
	{"agent_ask", bAgentAsk},
	{"agent_clear", bAgentClear},
	{"try_ai_log", bTryAILog},
	{"_try_ai_eval", bTryAIEval},

	// AI — Swarms
	{"swarm_agent", bSwarmAgent},
	{"ai_swarm", bAiSwarm},
	{"ai_swarm_trace", bAiSwarmTrace},

	// AI — Vision
	{"ai_vision", bAiVision},

	// Test Assertions
	{"assert", bAssert},
	{"assert_eq", bAssertEq},
	{"assert_not_eq", bAssertNotEq},
	{"assert_lt", bAssertLt},
	{"assert_gt", bAssertGt},
	{"assert_error", bAssertError},
	{"assert_near", bAssertNear},
	{"assert_contains", bAssertContains},

	// Error handling
	{"raise", bRaise},
}

// ---- Helpers ----

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

func bRaise(args ...Object) Object {
	if len(args) != 1 {
		return err("raise expects 1 argument (message)")
	}
	if args[0] == nil {
		return err("nil")
	}
	return err(args[0].Inspect())
}

func FormatMsg(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
