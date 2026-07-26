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

	"github.com/harry/pulse/pkg/ast"
)

type ObjectType string

const (
	INTEGER          ObjectType = "INTEGER"
	FLOAT                       = "FLOAT"
	STRING                      = "STRING"
	BOOLEAN                     = "BOOLEAN"
	NIL                         = "NIL"
	FUNCTION                    = "FUNCTION"
	COMPILED_FUNCTION           = "COMPILED_FUNCTION"
	LIST                        = "LIST"
	MAP                         = "MAP"
	ERROR                       = "ERROR"
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
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type BuiltinInfo struct {
	Name string
	Fn   func(args ...Object) Object
}

func (bi *BuiltinInfo) Type() ObjectType { return "BUILTIN" }
func (bi *BuiltinInfo) Inspect() string  { return "builtin: " + bi.Name }

type CompiledFunction struct {
	Instructions interface{}
	NumLocals    int
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION }
func (cf *CompiledFunction) Inspect() string  { return "compiled function" }

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
	{"sort", bSort},
	{"range", bRange},

	// List — higher order (VM can't use these yet, added for completeness)
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
}

// ---- IO ----

func bPrint(args ...Object) Object {
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
		return err("read_file erwartet 1 Argument (Pfad)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("read_file erwartet einen String als Pfad")
	}
	data, e := os.ReadFile(s.Value)
	if e != nil {
		return err("read_file: " + e.Error())
	}
	return &String{Value: string(data)}
}

func bWriteFile(args ...Object) Object {
	if len(args) != 2 {
		return err("write_file erwartet 2 Argumente (Pfad, Inhalt)")
	}
	p, ok := args[0].(*String)
	c, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("write_file: Pfad und Inhalt müssen Strings sein")
	}
	if e := os.WriteFile(p.Value, []byte(c.Value), 0644); e != nil {
		return err("write_file: " + e.Error())
	}
	return NILOBJ
}

func bEnv(args ...Object) Object {
	if len(args) != 1 {
		return err("env erwartet 1 Argument (Name)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("env: Name muss String sein")
	}
	val := os.Getenv(name.Value)
	return &String{Value: val}
}

func bSleep(args ...Object) Object {
	if len(args) != 1 {
		return err("sleep erwartet 1 Argument (Millisekunden)")
	}
	ms, ok := ToInt(args[0])
	if !ok {
		return err("sleep: Millisekunden müssen Zahl sein")
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return NILOBJ
}

func bExec(args ...Object) Object {
	if len(args) != 1 {
		return err("exec erwartet 1 Argument (Befehl)")
	}
	cmd, ok := args[0].(*String)
	if !ok {
		return err("exec: Befehl muss ein String sein")
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
		return err("append_file erwartet 2 Argumente (Pfad, Inhalt)")
	}
	p, ok := args[0].(*String)
	c, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("append_file: Pfad und Inhalt müssen Strings sein")
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
		return err("read_lines erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("read_lines: Pfad muss String sein")
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
		return err("file_exists erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_exists: Pfad muss String sein")
	}
	_, e := os.Stat(p.Value)
	return NativeBoolToBoolean(e == nil)
}

func bFileDelete(args ...Object) Object {
	if len(args) != 1 {
		return err("file_delete erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_delete: Pfad muss String sein")
	}
	if e := os.Remove(p.Value); e != nil {
		return err("file_delete: " + e.Error())
	}
	return NILOBJ
}

func bFileMove(args ...Object) Object {
	if len(args) != 2 {
		return err("file_move erwartet 2 Argumente (Quelle, Ziel)")
	}
	src, ok := args[0].(*String)
	dst, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("file_move: Pfade müssen Strings sein")
	}
	if e := os.Rename(src.Value, dst.Value); e != nil {
		return err("file_move: " + e.Error())
	}
	return NILOBJ
}

func bFileCopy(args ...Object) Object {
	if len(args) != 2 {
		return err("file_copy erwartet 2 Argumente (Quelle, Ziel)")
	}
	src, ok := args[0].(*String)
	dst, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("file_copy: Pfade müssen Strings sein")
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
		return err("file_size erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_size: Pfad muss String sein")
	}
	info, e := os.Stat(p.Value)
	if e != nil {
		return err("file_size: " + e.Error())
	}
	return &Integer{Value: info.Size()}
}

func bFileType(args ...Object) Object {
	if len(args) != 1 {
		return err("file_type erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("file_type: Pfad muss String sein")
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
			return err("list_dir: Pfad muss String sein")
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
		return err("make_dir erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("make_dir: Pfad muss String sein")
	}
	if e := os.MkdirAll(p.Value, 0755); e != nil {
		return err("make_dir: " + e.Error())
	}
	return NILOBJ
}

func bRemoveDir(args ...Object) Object {
	if len(args) != 1 {
		return err("remove_dir erwartet 1 Argument (Pfad)")
	}
	p, ok := args[0].(*String)
	if !ok {
		return err("remove_dir: Pfad muss String sein")
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
			return err("path_join: alle Argumente müssen Strings sein")
		}
		parts[i] = s.Value
	}
	return &String{Value: filepath.Join(parts...)}
}

func bPathBase(args ...Object) Object {
	if len(args) != 1 {
		return err("path_base erwartet 1 Argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_base erwartet einen String")
	}
	return &String{Value: filepath.Base(s.Value)}
}

func bPathDir(args ...Object) Object {
	if len(args) != 1 {
		return err("path_dir erwartet 1 Argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_dir erwartet einen String")
	}
	return &String{Value: filepath.Dir(s.Value)}
}

func bPathExt(args ...Object) Object {
	if len(args) != 1 {
		return err("path_ext erwartet 1 Argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("path_ext erwartet einen String")
	}
	return &String{Value: filepath.Ext(s.Value)}
}

// ---- String ----

func bUpper(args ...Object) Object {
	if s, ok := strArg(args, "upper"); ok {
		return &String{Value: strings.ToUpper(s.Value)}
	}
	return err("upper erwartet einen String")
}

func bLower(args ...Object) Object {
	if s, ok := strArg(args, "lower"); ok {
		return &String{Value: strings.ToLower(s.Value)}
	}
	return err("lower erwartet einen String")
}

func bTrim(args ...Object) Object {
	if s, ok := strArg(args, "trim"); ok {
		return &String{Value: strings.TrimSpace(s.Value)}
	}
	return err("trim erwartet einen String")
}

func bSplit(args ...Object) Object {
	if len(args) != 2 {
		return err("split erwartet 2 Argumente")
	}
	s, ok := args[0].(*String)
	d, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("split: beide Argumente müssen Strings sein")
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
		return err("join erwartet 2 Argumente")
	}
	l, ok := args[0].(*List)
	d, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("join: List und String erwartet")
	}
	parts := make([]string, len(l.Elements))
	for i, e := range l.Elements {
		parts[i] = e.Inspect()
	}
	return &String{Value: strings.Join(parts, d.Value)}
}

func bContains(args ...Object) Object {
	if len(args) != 2 {
		return err("contains erwartet 2 Argumente")
	}
	switch c := args[0].(type) {
	case *String:
		if sub, ok := args[1].(*String); ok {
			return NativeBoolToBoolean(strings.Contains(c.Value, sub.Value))
		}
		return err("contains: Substring muss String sein")
	case *List:
		for _, e := range c.Elements {
			if ValuesEqual(e, args[1]) {
				return TRUE
			}
		}
		return FALSE
	}
	return err("contains erwartet String oder List")
}

// ---- List ----

func bLen(args ...Object) Object {
	if len(args) != 1 {
		return err("len erwartet 1 Argument")
	}
	switch a := args[0].(type) {
	case *String:
		return &Integer{Value: int64(len(a.Value))}
	case *List:
		return &Integer{Value: int64(len(a.Elements))}
	}
	return err("len nicht unterstützt")
}

func bPush(args ...Object) Object {
	if len(args) < 2 {
		return err("push erwartet mindestens 2 Argumente")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("push: erstes Argument muss List sein")
	}
	l.Elements = append(l.Elements, args[1:]...)
	return l
}

func bPop(args ...Object) Object {
	if len(args) != 1 {
		return err("pop erwartet 1 Argument")
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
		return err("at erwartet 2 Argumente")
	}
	idx, ok := ToInt(args[1])
	if !ok {
		return err("at: Index muss Zahl sein")
	}
	switch c := args[0].(type) {
	case *List:
		if idx < 0 || idx >= int64(len(c.Elements)) {
			return err("at: Index außerhalb")
		}
		return c.Elements[idx]
	case *String:
		if idx < 0 || idx >= int64(len(c.Value)) {
			return err("at: Index außerhalb")
		}
		return &String{Value: string(c.Value[idx])}
	}
	return err("at erwartet List oder String")
}

func bSort(args ...Object) Object {
	if len(args) != 1 {
		return err("sort erwartet 1 Argument")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("sort erwartet List")
	}
	sorted := make([]Object, len(l.Elements))
	copy(sorted, l.Elements)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Inspect() < sorted[j].Inspect()
	})
	return &List{Elements: sorted}
}

func bMap(args ...Object) Object {
	if len(args) != 2 {
		return err("map erwartet 2 Argumente")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("map: erstes Argument muss List sein")
	}
	result := make([]Object, len(l.Elements))
	for i, e := range l.Elements {
		result[i] = callOne(args[1], e)
	}
	return &List{Elements: result}
}

func bFilter(args ...Object) Object {
	if len(args) != 2 {
		return err("filter erwartet 2 Argumente")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("filter: erstes Argument muss List sein")
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
		return err("reduce erwartet 3 Argumente")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("reduce: erstes Argument muss List sein")
	}
	acc := args[2]
	for _, e := range l.Elements {
		acc = callTwo(args[1], acc, e)
	}
	return acc
}

func bEach(args ...Object) Object {
	if len(args) != 2 {
		return err("each erwartet 2 Argumente")
	}
	l, ok := args[0].(*List)
	if !ok {
		return err("each: erstes Argument muss List sein")
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
	return err("map/filter/each: Funktion nicht aufrufbar (nur builtins im VM-Modus)")
}

func callTwo(fn, a, b Object) Object {
	if bi, ok := fn.(*BuiltinInfo); ok {
		return bi.Fn(a, b)
	}
	return err("reduce: Funktion nicht aufrufbar")
}

func bRange(args ...Object) Object {
	if len(args) < 1 || len(args) > 3 {
		return err("range erwartet 1-3 Argumente")
	}
	var start, end, step int64
	step = 1

	switch len(args) {
	case 1:
		n, ok := ToInt(args[0])
		if !ok {
			return err("range: Argument muss Zahl sein")
		}
		start = 0
		end = n
	case 2:
		s, ok1 := ToInt(args[0])
		e, ok2 := ToInt(args[1])
		if !ok1 || !ok2 {
			return err("range: Argumente müssen Zahlen sein")
		}
		start = s
		end = e
	case 3:
		s, ok1 := ToInt(args[0])
		e, ok2 := ToInt(args[1])
		st, ok3 := ToInt(args[2])
		if !ok1 || !ok2 || !ok3 {
			return err("range: Argumente müssen Zahlen sein")
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
		return err("abs erwartet 1 Argument")
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
	return err("abs erwartet eine Zahl")
}

func bMin(args ...Object) Object {
	if len(args) < 2 {
		return err("min erwartet mindestens 2 Argumente")
	}
	f, ok := ToFloat(args[0])
	if !ok {
		return err("min: Argumente müssen Zahlen sein")
	}
	allInt := true
	for _, a := range args {
		if _, isI := a.(*Float); isI {
			allInt = false
		}
		af, ok := ToFloat(a)
		if !ok {
			return err("min: Argumente müssen Zahlen sein")
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
		return err("max erwartet mindestens 2 Argumente")
	}
	f, ok := ToFloat(args[0])
	if !ok {
		return err("max: Argumente müssen Zahlen sein")
	}
	allInt := true
	for _, a := range args {
		if _, isI := a.(*Float); isI {
			allInt = false
		}
		af, ok := ToFloat(a)
		if !ok {
			return err("max: Argumente müssen Zahlen sein")
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
		return err("pow erwartet 2 Argumente")
	}
	b, ok1 := ToFloat(args[0])
	e, ok2 := ToFloat(args[1])
	if !ok1 || !ok2 {
		return err("pow: Argumente müssen Zahlen sein")
	}
	return &Float{Value: math.Pow(b, e)}
}

func bSqrt(args ...Object) Object {
	if len(args) != 1 {
		return err("sqrt erwartet 1 Argument")
	}
	v, ok := ToFloat(args[0])
	if !ok {
		return err("sqrt erwartet eine Zahl")
	}
	if v < 0 {
		return err("sqrt: negative Zahl")
	}
	return &Float{Value: math.Sqrt(v)}
}

func bRound(args ...Object) Object {
	if len(args) != 1 {
		return err("round erwartet 1 Argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return v
	case *Float:
		return &Integer{Value: int64(math.Round(v.Value))}
	}
	return err("round erwartet eine Zahl")
}

// ---- Map ----

func bKeys(args ...Object) Object {
	if len(args) != 1 {
		return err("keys erwartet 1 Argument")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("keys erwartet eine Map")
	}
	keys := make([]Object, 0, len(m.Pairs))
	for k := range m.Pairs {
		keys = append(keys, &String{Value: k})
	}
	return &List{Elements: keys}
}

func bValues(args ...Object) Object {
	if len(args) != 1 {
		return err("values erwartet 1 Argument")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("values erwartet eine Map")
	}
	vals := make([]Object, 0, len(m.Pairs))
	for _, v := range m.Pairs {
		vals = append(vals, v)
	}
	return &List{Elements: vals}
}

func bGet(args ...Object) Object {
	if len(args) != 2 {
		return err("get erwartet 2 Argumente (Map/List, Key/Index)")
	}
	switch container := args[0].(type) {
	case *Map:
		key, ok := args[1].(*String)
		if !ok {
			return err("get: Map-Key muss ein String sein")
		}
		val, exists := container.Pairs[key.Value]
		if !exists {
			return NILOBJ
		}
		return val
	case *List:
		idx, ok := ToInt(args[1])
		if !ok {
			return err("get: Listen-Index muss eine Zahl sein")
		}
		if idx < 0 || idx >= int64(len(container.Elements)) {
			return NILOBJ
		}
		return container.Elements[idx]
	}
	return err("get erwartet eine Map oder List")
}

func bSet(args ...Object) Object {
	if len(args) != 3 {
		return err("set erwartet 3 Argumente (Map, Key, Value)")
	}
	m, ok := args[0].(*Map)
	if !ok {
		return err("set: erstes Argument muss eine Map sein")
	}
	key, ok := args[1].(*String)
	if !ok {
		return err("set: Key muss ein String sein")
	}
	m.Pairs[key.Value] = args[2]
	return m
}

// ---- Type checks ----

func bTypeOf(args ...Object) Object {
	if len(args) != 1 {
		return err("type_of erwartet 1 Argument")
	}
	return &String{Value: string(args[0].Type())}
}

func bIsNum(args ...Object) Object {
	if len(args) != 1 {
		return err("is_num erwartet 1 Argument")
	}
	_, i := args[0].(*Integer)
	_, f := args[0].(*Float)
	return NativeBoolToBoolean(i || f)
}

func bIsStr(args ...Object) Object {
	if len(args) != 1 {
		return err("is_str erwartet 1 Argument")
	}
	_, ok := args[0].(*String)
	return NativeBoolToBoolean(ok)
}

func bIsList(args ...Object) Object {
	if len(args) != 1 {
		return err("is_list erwartet 1 Argument")
	}
	_, ok := args[0].(*List)
	return NativeBoolToBoolean(ok)
}

func bIsMap(args ...Object) Object {
	if len(args) != 1 {
		return err("is_map erwartet 1 Argument")
	}
	_, ok := args[0].(*Map)
	return NativeBoolToBoolean(ok)
}

func bIsNil(args ...Object) Object {
	if len(args) != 1 {
		return err("is_nil erwartet 1 Argument")
	}
	return NativeBoolToBoolean(args[0] == NILOBJ)
}

// ---- Conversion ----

func bToStr(args ...Object) Object {
	if len(args) != 1 {
		return err("to_str erwartet 1 Argument")
	}
	return &String{Value: args[0].Inspect()}
}

func bToNum(args ...Object) Object {
	if len(args) != 1 {
		return err("to_num erwartet 1 Argument")
	}
	switch v := args[0].(type) {
	case *Integer:
		return v
	case *Float:
		return v
	case *String:
		f, e := strconv.ParseFloat(v.Value, 64)
		if e != nil {
			return err("to_num: '" + v.Value + "' ist keine Zahl")
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
	return err("to_num nicht möglich")
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
	if len(args) != 1 {
		return err("http_get erwartet 1 Argument (URL)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_get: URL muss ein String sein")
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
	if len(args) < 1 || len(args) > 2 {
		return err("http_post erwartet 1-2 Argumente (URL, Body?)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_post: URL muss ein String sein")
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
		return err("parse_json erwartet 1 Argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("parse_json erwartet einen String")
	}
	return jsonToObject(s.Value)
}

func bToJSON(args ...Object) Object {
	if len(args) != 1 {
		return err("to_json erwartet 1 Argument")
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
	if len(args) != 2 {
		return err("tcp_listen erwartet 2 Argumente (Host, Port)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_listen: Host muss String sein")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_listen: Port muss Zahl sein")
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
	if len(args) != 2 {
		return err("tcp_connect erwartet 2 Argumente (Host, Port)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_connect: Host muss String sein")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_connect: Port muss Zahl sein")
	}
	addr := fmt.Sprintf("%s:%d", host.Value, port)
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
		return err("tcp_accept erwartet 1 Argument (Listener)")
	}
	ln, ok := args[0].(*TcpListener)
	if !ok {
		return err("tcp_accept erwartet einen TCP-Listener")
	}
	connMu.Lock()
	listener, exists := listeners[ln.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_accept: Listener existiert nicht")
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
		return err("tcp_read erwartet 1 Argument (Verbindung)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_read erwartet eine TCP-Verbindung")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_read: Verbindung existiert nicht")
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
		return err("tcp_write erwartet 2 Argumente (Verbindung, Daten)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_write erwartet eine TCP-Verbindung")
	}
	data, ok := args[1].(*String)
	if !ok {
		return err("tcp_write: Daten müssen String sein")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_write: Verbindung existiert nicht")
	}
	_, e := c.Write([]byte(data.Value))
	if e != nil {
		return err("tcp_write: " + e.Error())
	}
	return NILOBJ
}

func bTcpClose(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_close erwartet 1 Argument")
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
		return err("tcp_close: ungültiger Typ")
	}
	return NILOBJ
}

// ---- Regex ----

func bRegexMatch(args ...Object) Object {
	if len(args) != 2 {
		return err("regex_match erwartet 2 Argumente (Pattern, Text)")
	}
	pat, ok := args[0].(*String)
	txt, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("regex_match: Pattern und Text müssen Strings sein")
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
		return err("regex_replace erwartet 3 Argumente (Pattern, Ersatz, Text)")
	}
	pat, ok := args[0].(*String)
	repl, ok2 := args[1].(*String)
	txt, ok3 := args[2].(*String)
	if !ok || !ok2 || !ok3 {
		return err("regex_replace: Alle Argumente müssen Strings sein")
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
		return err("format_time erwartet 1-2 Argumente (Timestamp, Format?)")
	}
	ts, ok := ToInt(args[0])
	if !ok {
		return err("format_time: Timestamp muss Zahl sein")
	}
	layout := "2006-01-02 15:04:05"
	if len(args) >= 2 {
		s, ok := args[1].(*String)
		if !ok {
			return err("format_time: Format muss String sein")
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
		return err("random_range erwartet 2 Argumente (Min, Max)")
	}
	min, ok1 := ToInt(args[0])
	max, ok2 := ToInt(args[1])
	if !ok1 || !ok2 {
		return err("random_range: Min und Max müssen Zahlen sein")
	}
	if min >= max {
		return err("random_range: Min muss kleiner als Max sein")
	}
	return &Integer{Value: min + rand.Int63n(max-min)}
}

// ---- Helpers ----

// ---- Encoding ----

func bBase64Encode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64_encode erwartet 1 Argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64_encode erwartet einen String")
	}
	return &String{Value: base64.StdEncoding.EncodeToString([]byte(s.Value))}
}

func bBase64Decode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64_decode erwartet 1 Argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64_decode erwartet einen String")
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
