package cache

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
)

const (
	magic = "PIPEBC"
	// version comes from the compiler so that any bytecode-semantics change
	// (new opcodes, changed symbol/global allocation, ...) invalidates every
	// existing .pipec cache instead of feeding stale bytecode to the VM.
	version = compiler.CacheVersion
)

// LoadOrCompile returns the bytecode for a source file, reusing a cached
// copy when the file and every module it imports (transitively) are
// unchanged. The second return value reports whether the result came from
// the cache. When compiling, the cache is refreshed so repeated runs of the
// same file skip compilation entirely.
func LoadOrCompile(filePath string) (*compiler.Bytecode, bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, err
	}

	deps, derr := depsHash(filePath, data)
	if derr == nil {
		cachePath := filePath + "c"
		if bc, lerr := loadCache(cachePath, deps); lerr == nil && bc != nil {
			return bc, true, nil
		}
	} else {
		// The dependency graph could not be computed (e.g. an unresolvable
		// import). Fall back to compiling directly; the compiler then reports
		// the real error to the caller.
		deps = ""
	}

	bc, cerr := compileSource(filePath, data)
	if cerr != nil {
		return nil, false, cerr
	}
	if deps != "" {
		_ = writeCache(filePath+"c", deps, bc)
	}
	return bc, false, nil
}

func compileSource(filePath string, data []byte) (*compiler.Bytecode, error) {
	program, err := object.ParseContent(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filePath, err)
	}
	c := compiler.NewWithFile(filePath)
	if err := c.Compile(program); err != nil {
		return nil, err
	}
	return c.Bytecode(), nil
}

// depsHash returns a stable hash over the file and every module it imports
// (transitively). Because the compiler embeds imported modules into the
// bytecode at compile time, the cache must be invalidated when any dependency
// changes, not just the top-level file. Imports are resolved the same way the
// compiler resolves them, using each importing file as the resolution context.
func depsHash(filePath string, data []byte) (string, error) {
	visited := map[string]bool{}
	contents := map[string][]byte{}
	var walk func(path string) error
	walk = func(path string) error {
		if visited[path] {
			return nil
		}
		visited[path] = true
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contents[path] = content

		prog, perr := object.ParseContent(string(content))
		if perr != nil {
			// An unparseable dependency still contributes to the hash, but its
			// own imports cannot be enumerated.
			return nil
		}
		for _, stmt := range prog.Statements {
			is, ok := stmt.(*ast.ImportStatement)
			if !ok {
				continue
			}
			resPath, _, rerr := object.ResolveImportFrom(is.Path, path)
			if rerr != nil {
				return rerr
			}
			if err := walk(resPath); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(filePath); err != nil {
		return "", err
	}

	paths := make([]string, 0, len(contents))
	for p := range contents {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	h := sha256.New()
	for _, p := range paths {
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write(contents[p])
		h.Write([]byte{0})
	}
	// The compiler bakes each builtin's position in object.Builtins directly
	// into the bytecode as its BuiltinScope index (see compiler.resolveBuiltin).
	// Adding, removing, or reordering a builtin anywhere in that table shifts
	// every later builtin's index, so a .pipec compiled against an older table
	// layout would otherwise still look "valid" (same source, same
	// CacheVersion) while resolving OpGetBuiltin to the wrong function. Mixing
	// the ordered builtin names into the cache key makes that class of staleness
	// self-invalidating instead of depending on a developer remembering to bump
	// CacheVersion for a change that doesn't touch bytecode format at all.
	for _, b := range object.Builtins {
		h.Write([]byte(b.Name))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:16]), nil
}

func loadCache(path, deps string) (*compiler.Bytecode, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	magicBuf := make([]byte, len(magic))
	if _, err := f.Read(magicBuf); err != nil {
		return nil, err
	}
	if string(magicBuf) != magic {
		return nil, fmt.Errorf("invalid cache magic")
	}

	var ver byte
	binary.Read(f, binary.BigEndian, &ver)
	if ver != version {
		return nil, fmt.Errorf("cache version mismatch")
	}

	hashBuf := make([]byte, 16)
	if _, err := f.Read(hashBuf); err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", hashBuf) != deps {
		return nil, fmt.Errorf("dependencies changed")
	}

	var numConstants uint32
	binary.Read(f, binary.BigEndian, &numConstants)
	constants := make([]object.Object, numConstants)
	for i := uint32(0); i < numConstants; i++ {
		var typ byte
		binary.Read(f, binary.BigEndian, &typ)
		switch typ {
		case 1: // Integer
			var v int64
			binary.Read(f, binary.BigEndian, &v)
			constants[i] = &object.Integer{Value: v}
		case 2: // Float
			var v float64
			binary.Read(f, binary.BigEndian, &v)
			constants[i] = &object.Float{Value: v}
		case 3: // String
			var length uint16
			binary.Read(f, binary.BigEndian, &length)
			buf := make([]byte, length)
			f.Read(buf)
			constants[i] = &object.String{Value: string(buf)}
		case 4: // CompiledFunction
			var numLocals int32
			binary.Read(f, binary.BigEndian, &numLocals)
			var insLen uint32
			binary.Read(f, binary.BigEndian, &insLen)
			ins := make([]byte, insLen)
			f.Read(ins)
			var numLines uint32
			binary.Read(f, binary.BigEndian, &numLines)
			lines := make([]int, numLines)
			for j := uint32(0); j < numLines; j++ {
				var ln int32
				binary.Read(f, binary.BigEndian, &ln)
				lines[j] = int(ln)
			}
			constants[i] = &object.CompiledFunction{
				Instructions: compiler.Instructions(ins),
				Lines:        lines,
				NumLocals:    int(numLocals),
			}
		}
	}

	var insLen uint32
	binary.Read(f, binary.BigEndian, &insLen)
	instructions := make(compiler.Instructions, insLen)
	f.Read(instructions)

	var numLines uint32
	binary.Read(f, binary.BigEndian, &numLines)
	lines := make([]int, numLines)
	for j := uint32(0); j < numLines; j++ {
		var ln int32
		binary.Read(f, binary.BigEndian, &ln)
		lines[j] = int(ln)
	}

	return &compiler.Bytecode{
		Instructions: instructions,
		Lines:        lines,
		Constants:    constants,
	}, nil
}

func writeCache(path, deps string, bc *compiler.Bytecode) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCacheData(f, deps, bc)
}

func writeCacheData(f *os.File, deps string, bc *compiler.Bytecode) error {
	f.Write([]byte(magic))
	f.Write([]byte{version})

	hashDecoded, err := hex.DecodeString(deps)
	if err != nil || len(hashDecoded) != 16 {
		return fmt.Errorf("invalid dependency hash %q", deps)
	}
	f.Write(hashDecoded)

	numConstants := uint32(len(bc.Constants))
	binary.Write(f, binary.BigEndian, numConstants)
	for _, obj := range bc.Constants {
		switch v := obj.(type) {
		case *object.Integer:
			f.Write([]byte{1})
			binary.Write(f, binary.BigEndian, v.Value)
		case *object.Float:
			f.Write([]byte{2})
			binary.Write(f, binary.BigEndian, v.Value)
		case *object.String:
			f.Write([]byte{3})
			b := []byte(v.Value)
			binary.Write(f, binary.BigEndian, uint16(len(b)))
			f.Write(b)
		case *object.CompiledFunction:
			f.Write([]byte{4})
			binary.Write(f, binary.BigEndian, int32(v.NumLocals))
			if ins, ok := v.Instructions.(compiler.Instructions); ok {
				binary.Write(f, binary.BigEndian, uint32(len(ins)))
				f.Write(ins)
			} else {
				binary.Write(f, binary.BigEndian, uint32(0))
			}
			binary.Write(f, binary.BigEndian, uint32(len(v.Lines)))
			for _, ln := range v.Lines {
				binary.Write(f, binary.BigEndian, int32(ln))
			}
		}
	}

	insLen := uint32(len(bc.Instructions))
	binary.Write(f, binary.BigEndian, insLen)
	f.Write(bc.Instructions)

	binary.Write(f, binary.BigEndian, uint32(len(bc.Lines)))
	for _, ln := range bc.Lines {
		binary.Write(f, binary.BigEndian, int32(ln))
	}

	return nil
}
