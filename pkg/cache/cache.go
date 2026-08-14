package cache

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/lexer"
	"github.com/MachuraHarry/pipe/pkg/object"
	"github.com/MachuraHarry/pipe/pkg/parser"
)

const (
	magic   = "PIPEBC"
	version = byte(2)
)

func hashSource(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:16])
}

func LoadOrCompile(filePath string) (*compiler.Bytecode, bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, err
	}

	srcHash := hashSource(data)
	cachePath := filePath + "c"

	if info, err := os.Stat(cachePath); err == nil && info.Size() > 0 {
		if bc, err := loadCache(cachePath, data); err == nil && bc != nil {
			return bc, true, nil
		}
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, false, fmt.Errorf("%s: parse errors: %v", filePath, p.Errors())
	}

	c := compiler.NewWithFile(filePath)
	if err := c.Compile(program); err != nil {
		return nil, false, err
	}

	bc := c.Bytecode()
	_ = srcHash

	return bc, false, nil
}

func loadCache(path string, sourceData []byte) (*compiler.Bytecode, error) {
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
	if fmt.Sprintf("%x", hashBuf) != hashSource(sourceData) {
		return nil, fmt.Errorf("source changed")
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

func saveCache(path string, srcHash string, bc *compiler.Bytecode) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCacheData(f, srcHash, bc)
}

func WriteCache(path string, bc *compiler.Bytecode) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCacheData(f, "00000000000000000000000000000000", bc)
}

func writeCacheData(f *os.File, srcHash string, bc *compiler.Bytecode) error {
	f.Write([]byte(magic))
	f.Write([]byte{version})

	var hashDecoded [16]byte
	fmt.Sscanf(srcHash, "%x", &hashDecoded)
	f.Write(hashDecoded[:])

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
