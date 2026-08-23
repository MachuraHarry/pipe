package vm

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
)

// debugTick enables the instruction sampler in Run(); set via
// PIPE_VM_TICK=1. Temporary diagnostics aid.
var debugTick = os.Getenv("PIPE_VM_TICK") == "1"

const (
	StackSize = 2048
	// operandHeadroom keeps a few free slots after the recursion guard fires
	// so the innermost caller can finish unwinding (try/catch handling)
	// without tripping the operand stack in the same breath.
	operandHeadroom = 32
)

type Frame struct {
	closure      *object.Closure
	ip           int
	basePointer  int
	savedSp      int
	instructions compiler.Instructions
	lines        []int // source line per instruction byte; may be nil
}

func (f *Frame) lineAt(ip int) int {
	if f.lines != nil && ip >= 0 && ip < len(f.lines) {
		return f.lines[ip]
	}
	return 0
}

func New(bc *compiler.Bytecode) *VM {
	stack := make([]object.Object, StackSize)
	globals := make([]object.Object, 65536)
	frames := make([]*Frame, object.MaxCallDepth)

	mainFn := &object.CompiledFunction{
		Instructions: bc.Instructions,
	}
	mainClosure := &object.Closure{
		Fn:   mainFn,
		Free: []object.Object{},
	}
	mainFrame := &Frame{
		closure:      mainClosure,
		ip:           0,
		basePointer:  0,
		instructions: bc.Instructions,
		lines:        bc.Lines,
	}

	frames[0] = mainFrame

	vm := &VM{
		constants:  bc.Constants,
		globals:    globals,
		stack:      stack,
		sp:         0,
		frames:     frames,
		frameIndex: 0,
		sourceFile: bc.SourceFile,
	}

	return vm
}

// CallUserFunction satisfies object.UserFunctionExecutor: builtins such as
// map/filter/reduce call back into this VM. It is safe to call while this VM
// is the only executor of its operand stack, which holds because every
// parallel task runs on its own child VM.
func (vm *VM) CallUserFunction(fn object.Object, args ...object.Object) object.Object {
	return vm.callUserFunction(fn, args...)
}

// SpawnUserFunction satisfies object.UserFunctionSpawner: `go` and `spawn`
// launch a closure on a fresh child VM and get back its Future.
func (vm *VM) SpawnUserFunction(fn object.Object, args ...object.Object) *object.Future {
	cl, ok := fn.(*object.Closure)
	if !ok {
		return nil
	}
	return vm.spawnClosure(cl, args)
}

// spawnClosure launches a closure on a child VM in a goroutine and returns a
// Future that resolves to its result.
func (vm *VM) spawnClosure(fn *object.Closure, args []object.Object) *object.Future {
	future := object.NewFuture()
	globals := vm.snapshotGlobals()
	go func() {
		child := vm.newSpawnVM(fn, args, globals)
		future.Val = child.executeFrame()
		close(future.Done)
	}()
	return future
}

type VM struct {
	constants  []object.Object
	globals    []object.Object
	stack      []object.Object
	sp         int
	frames     []*Frame
	frameIndex int
	curLine    int
	tick       uint64
	sourceFile string
	// pendingError is the first error produced since the last try/catch
	// boundary. The tree-walker aborts at an uncaught error; the VM cannot
	// unwind frames, so it records the error here, lets try/catch (OpCheckError)
	// clear it, and reports it when the program ends (OpHalt / end of Run).
	pendingError *object.Error
	// TestFailed is set when a compiled `test` block produced an error. The
	// test runner checks it to fail the file, mirroring the tree-walker's
	// testFailed flag.
	TestFailed bool
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.frameIndex]
}

// newError builds a positioned runtime error object for a VM failure.
func (vm *VM) newError(code, format string, a ...interface{}) *object.Error {
	prefix := ""
	if vm.sourceFile != "" {
		prefix = vm.sourceFile + ": "
	}
	return &object.Error{
		Message: prefix + fmt.Sprintf(code+": "+format, a...),
		Code:    code,
		File:    vm.sourceFile,
		Line:    vm.curLine,
	}
}

func (vm *VM) push(obj object.Object) {
	if vm.sp >= StackSize {
		panic("stack overflow")
	}
	vm.stack[vm.sp] = obj
	vm.sp++
	if errObj, ok := obj.(*object.Error); ok {
		vm.pendingError = errObj
	}
}

// reportPending returns the first uncaught error of this VM, if any. It is
// called only at top-level exit points of Run (never from executeFrame, which
// serves nested closures whose errors are handled by an enclosing try/catch).
func (vm *VM) reportPending() error {
	if vm.pendingError != nil {
		return vm.pendingError
	}
	return nil
}

func (vm *VM) pop() object.Object {
	if vm.sp == 0 {
		return object.NILOBJ
	}
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *VM) peek() object.Object {
	if vm.sp == 0 {
		return object.NILOBJ
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) readUint16() uint16 {
	frame := vm.currentFrame()
	val := compiler.ReadUint16(frame.instructions, frame.ip)
	frame.ip += 2
	return val
}

// runTestAbortIfError implements OpTestAbortIfError: when the body of a
// compiled `test` block has produced an error, the test aborts early (matching
// the tree-walker's block short-circuit) and jumps to the test verdict with the
// error as the result.
func (vm *VM) runTestAbortIfError(ins compiler.Instructions, frame *Frame) {
	target := compiler.ReadUint16(ins, frame.ip)
	frame.ip += 2
	if vm.pendingError != nil {
		vm.push(vm.pendingError)
		frame.ip = int(target)
	}
}

// runTestResult implements OpTestResult: it reports a compiled `test` block as
// PASS or FAIL and records failures on the VM so the test runner can fail the
// file. The test name is read from the constants table.
func (vm *VM) runTestResult(ins compiler.Instructions, frame *Frame) {
	idx := compiler.ReadUint16(ins, frame.ip)
	frame.ip += 2
	val := vm.pop()
	name := ""
	if nameObj, ok := vm.constants[idx].(*object.String); ok {
		name = nameObj.Value
	}
	vm.pendingError = nil
	if _, isErr := val.(*object.Error); isErr {
		fmt.Printf("  FAIL %s (%s)\n", name, val.Inspect())
		vm.TestFailed = true
		vm.push(object.NILOBJ)
	} else {
		fmt.Printf("  PASS %s\n", name)
		vm.push(val)
	}
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok && s == "stack overflow" {
				err = fmt.Errorf("stack overflow: recursion too deep (operand stack exhausted)")
				return
			}
			panic(r)
		}
	}()
	for {
		frame := vm.currentFrame()
		ins := frame.instructions

		if debugTick {
			vm.tick++
			if vm.tick%300000 == 0 {
				if cl := frame.closure; cl != nil {
					fmt.Fprintf(os.Stderr, "TICK line=%d ip=%d op=%d sp=%d fr=%d id=ins%d/loc%d/l%d", frame.lineAt(frame.ip), frame.ip, ins[frame.ip], vm.sp, vm.frameIndex, len(frame.instructions), cl.Fn.NumLocals, len(cl.Fn.Lines))
					if len(frame.instructions) == 687 && frame.ip >= 500 {
						l9 := vm.stack[frame.basePointer+9]
						l10 := vm.stack[frame.basePointer+10]
						fmt.Fprintf(os.Stderr, " L9=%v L10=%v bp=%d", l9, l10, frame.basePointer)
					}
					if len(frame.instructions) == 687 && frame.ip == 663 {
						fmt.Fprintf(os.Stderr, " | vi=%v vr=%v", vm.stack[frame.basePointer+13], vm.stack[frame.basePointer+5])
					}
					fmt.Fprintln(os.Stderr)
				}
			}
		}

		if frame.ip >= len(ins) {
			if err := vm.reportPending(); err != nil {
				return err
			}
			break
		}

		op := compiler.Opcode(ins[frame.ip])
		vm.curLine = frame.lineAt(frame.ip)
		frame.ip++

		switch op {
		case compiler.OpConstant:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.constants[idx])

		case compiler.OpTrue:
			vm.push(object.TRUE)

		case compiler.OpFalse:
			vm.push(object.FALSE)

		case compiler.OpNil:
			vm.push(object.NILOBJ)

		case compiler.OpPop:
			vm.pop()

		case compiler.OpDup:
			if vm.sp > 0 {
				vm.push(vm.stack[vm.sp-1])
			}

		case compiler.OpAdd, compiler.OpSub, compiler.OpMul, compiler.OpDiv, compiler.OpMod, compiler.OpPow:
			right := vm.pop()
			left := vm.pop()
			result := vm.binaryOp(op, left, right)
			vm.push(result)

		case compiler.OpEqual, compiler.OpNotEqual, compiler.OpGreater, compiler.OpLess,
			compiler.OpGte, compiler.OpLte:
			right := vm.pop()
			left := vm.pop()
			result := vm.compareOp(op, left, right)
			vm.push(result)

		case compiler.OpConcat:
			vm.concatOp()

		case compiler.OpMinus:
			val := vm.pop()
			switch v := val.(type) {
			case *object.Integer:
				vm.push(&object.Integer{Value: -v.Value})
			case *object.Float:
				vm.push(&object.Float{Value: -v.Value})
			default:
				return vm.newError("E005", "cannot negate a %s with '-'", val.Type())
			}

		case compiler.OpNot:
			val := vm.pop()
			vm.push(object.NativeBoolToBoolean(!object.IsTruthy(val)))

		case compiler.OpJump:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip = int(target)

		case compiler.OpJumpNotTruthy:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			if !object.IsTruthy(vm.pop()) {
				frame.ip = int(target)
			}

		case compiler.OpJumpBackward:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip = int(target)

		case compiler.OpGetGlobal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.globals[idx])

		case compiler.OpSetGlobal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.globals[idx] = vm.pop()

		case compiler.OpGetLocal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.stack[frame.basePointer+int(idx)])

		case compiler.OpSetLocal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.stack[frame.basePointer+int(idx)] = vm.pop()

		case compiler.OpGetBuiltin:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			if int(idx) >= len(object.Builtins) {
				return vm.newError("E004", "unknown builtin function: %d", idx)
			}
			vm.push(&object.BuiltinInfo{
				Name: object.Builtins[idx].Name,
				Fn:   object.Builtins[idx].Fn,
			})

		case compiler.OpGetFree:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(frame.closure.Free[int(idx)])

		case compiler.OpCheckError:
			val := vm.peek()
			if _, isErr := val.(*object.Error); isErr {
				vm.pendingError = nil
				vm.push(object.TRUE)
			} else if vm.pendingError != nil {
				// The error's value was discarded before the check (e.g. an
				// expression statement inside try); surface it so the catch
				// binding can convert it with OpErrorToString.
				vm.stack[vm.sp-1] = vm.pendingError
				vm.pendingError = nil
				vm.push(object.TRUE)
			} else {
				vm.push(object.FALSE)
			}

		case compiler.OpTestAbortIfError:
			vm.runTestAbortIfError(ins, frame)

		case compiler.OpTestResult:
			vm.runTestResult(ins, frame)

		case compiler.OpTryAIFix:
			src := vm.pop()
			srcStr, isStr := src.(*object.String)
			if !isStr || object.TryAIEvalFn == nil {
				vm.push(object.NILOBJ)
				continue
			}
			fixed := object.TryAIEvalFn(srcStr.Value)
			vm.push(fixed)

		case compiler.OpErrorToString:
			vm.pendingError = nil
			val := vm.pop()
			if err, isErr := val.(*object.Error); isErr {
				vm.push(&object.String{Value: err.Message})
			} else {
				vm.push(object.NILOBJ)
			}

		case compiler.OpCall:
			numArgs := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			vm.callFunction(numArgs)

		case compiler.OpSpawn:
			numArgs := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			vm.spawnCall(numArgs)

		case compiler.OpReturn:
			frame := vm.currentFrame()
			vm.sp = frame.savedSp
			vm.frameIndex--
			if vm.frameIndex < 0 {
				if err := vm.reportPending(); err != nil {
					return err
				}
				return nil
			}
			vm.push(object.NILOBJ)

		case compiler.OpReturnValue:
			frame := vm.currentFrame()
			returnVal := vm.pop()
			vm.sp = frame.savedSp
			vm.frameIndex--
			if vm.frameIndex < 0 {
				if err := vm.reportPending(); err != nil {
					return err
				}
				return nil
			}
			vm.push(returnVal)

		case compiler.OpClosure:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			numFree := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			fn, ok := vm.constants[idx].(*object.CompiledFunction)
			if !ok {
				return vm.newError("E004", "not a CompiledFunction at index %d", idx)
			}
			free := make([]object.Object, numFree)
			for i := numFree - 1; i >= 0; i-- {
				free[i] = vm.pop()
			}
			closure := &object.Closure{
				Fn:       fn,
				Free:     free,
				Executor: vm,
			}
			vm.push(closure)

		case compiler.OpList:
			numElems := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			elems := make([]object.Object, numElems)
			for i := numElems - 1; i >= 0; i-- {
				elems[i] = vm.pop()
			}
			vm.push(&object.List{Elements: elems})

		case compiler.OpMap:
			numPairs := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			vals := make([]object.Object, numPairs)
			for i := numPairs - 1; i >= 0; i-- {
				vals[i] = vm.pop()
			}
			pairs := make(map[string]object.Object)
			for i := 0; i < numPairs; i++ {
				ki := compiler.ReadUint16(ins, frame.ip)
				frame.ip += 2
				keyObj := vm.constants[ki]
				if ks, ok := keyObj.(*object.String); ok {
					pairs[ks.Value] = vals[i]
				}
			}
			vm.push(&object.Map{Pairs: pairs})

		case compiler.OpStruct:
			numFields := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			fieldNames := make([]string, numFields)
			for i := 0; i < numFields; i++ {
				nameIdx := compiler.ReadUint16(ins, frame.ip)
				frame.ip += 2
				fieldNames[i] = vm.constants[nameIdx].(*object.String).Value
			}
			vals := make([]object.Object, numFields)
			for i := numFields - 1; i >= 0; i-- {
				vals[i] = vm.pop()
			}
			defObj := vm.pop()
			def, ok := defObj.(*object.StructDef)
			if !ok {
				return vm.newError("E004", "expected struct definition, got %T", defObj)
			}
			inst := &object.StructInstance{
				Def:    def,
				Values: make(map[string]object.Object),
			}
			for i, fn := range fieldNames {
				inst.Values[fn] = vals[i]
			}
			vm.push(inst)

		case compiler.OpSelect:
			numCases := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2

			// Collect channels from stack
			channels := make([]object.Object, numCases)
			for i := numCases - 1; i >= 0; i-- {
				channels[i] = vm.pop()
			}

			// Try to find a ready channel (non-blocking check)
			// In a real implementation this would use Go's select with goroutines
			// For now, try each channel's try_recv
			var result object.Object = object.NILOBJ
			found := false
			for _, ch := range channels {
				if chanObj, ok := ch.(*object.Channel); ok {
					val := chanObj.TryRecv()
					if val != nil && val.Type() != object.NIL {
						result = val
						found = true
						break
					}
				}
			}

			if !found {
				// No channel ready - push nil, VM will skip body
				vm.push(object.NILOBJ)
			} else {
				vm.push(result)
			}

			// Skip case bodies (they're compiled but we handle them here)
			// Each case body is compiled with OpCall, we need to skip them
			// and execute the matched one
			for i := 0; i < numCases; i++ {
				// Skip the body instructions
				for {
					op := compiler.Opcode(ins[frame.ip])
					frame.ip++
					if op == compiler.OpPop {
						break
					}
					// Skip operand bytes
					switch op {
					case compiler.OpConstant, compiler.OpGetGlobal, compiler.OpSetGlobal,
						compiler.OpGetLocal, compiler.OpSetLocal, compiler.OpGetBuiltin,
						compiler.OpGetFree, compiler.OpClosure, compiler.OpCall,
						compiler.OpJump, compiler.OpJumpNotTruthy, compiler.OpStruct,
						compiler.OpList, compiler.OpMap, compiler.OpDot:
						frame.ip += 2
					}
				}
			}

			if !found {
				// Push nil for default case behavior
				vm.push(object.NILOBJ)
			}

		case compiler.OpDot:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			field := vm.constants[idx].(*object.String).Value
			obj := vm.pop()
			if obj == nil {
				vm.push(vm.newError("E006", "cannot use .%s on nil", field))
				continue
			}
			switch m := obj.(type) {
			case *object.StructInstance:
				if val, ok := m.Values[field]; ok {
					vm.push(val)
				} else {
					vm.push(object.NILOBJ)
				}
			case *object.Map:
				if val, ok := m.Pairs[field]; ok {
					vm.push(val)
				} else {
					vm.push(object.NILOBJ)
				}
			case *object.Error:
				if field == "message" {
					vm.push(&object.String{Value: m.Message})
				} else {
					vm.push(m)
				}
			default:
				vm.push(vm.newError("E006", "cannot use .%s on %s", field, obj.Type()))
			}

		case compiler.OpHalt:
			if err := vm.reportPending(); err != nil {
				return err
			}
			return nil

		default:
			return vm.newError("E004", "unknown opcode: %d", op)
		}
	}

	return nil
}

func (vm *VM) callFunction(numArgs int) {
	callee := vm.stack[vm.sp-1-numArgs]

	switch fn := callee.(type) {
	case *object.Closure:
		inst, ok := fn.Fn.Instructions.(compiler.Instructions)
		if !ok {
			vm.pop()
			vm.push(vm.newError("E004", "invalid compiled function instructions"))
			return
		}

		basePtr := vm.sp - numArgs
		savedSp := basePtr - 1

		frame := &Frame{
			closure:      fn,
			ip:           0,
			basePointer:  basePtr,
			savedSp:      savedSp,
			instructions: inst,
			lines:        fn.Fn.Lines,
		}

		vm.frameIndex++
		if vm.frameIndex >= object.MaxCallDepth || vm.sp >= StackSize-operandHeadroom {
			// Reject the call with a catchable error object instead of
			// panicking; try/catch in the VM (OpCheckError) can handle it.
			// The frame limit bounds recursion depth; the operand-space check
			// catches deep calls (many args per frame) that would exhaust the
			// operand stack before the frame limit is reached.
			vm.frameIndex--
			vm.sp = basePtr
			vm.stack[basePtr-1] = vm.newError("E008", "call stack depth exceeded (%d)", object.MaxCallDepth)
			vm.pendingError = vm.stack[basePtr-1].(*object.Error)
			return
		}
		vm.frames[vm.frameIndex] = frame

		localsNeeded := basePtr + fn.Fn.NumLocals
		if vm.sp < localsNeeded {
			vm.sp = localsNeeded
		}

	case *object.BuiltinInfo:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		if !object.IsAwaitBuiltin(fn) {
			for i := range args {
				args[i] = object.EnsureResolved(args[i])
			}
		}
		result := fn.Fn(args...)
		vm.pop()
		vm.push(result)

	case *object.StructDef:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		inst := &object.StructInstance{
			Def:    fn,
			Values: make(map[string]object.Object),
		}
		for k, v := range fn.Defaults {
			inst.Values[k] = v
		}
		for i, arg := range args {
			if i < len(fn.Fields) {
				inst.Values[fn.Fields[i]] = arg
			}
		}
		vm.pop()
		vm.push(inst)

	default:
		vm.pop()
		vm.push(vm.newError("E004", "not a function: %s", callee.Type()))
		return
	}
}

func (vm *VM) spawnCall(numArgs int) {
	callee := vm.stack[vm.sp-1-numArgs]

	switch fn := callee.(type) {
	case *object.BuiltinInfo:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		for i := range args {
			args[i] = object.EnsureResolved(args[i])
		}
		future := object.NewFuture()
		vm.pop()
		vm.push(future)
		go func() {
			result := fn.Fn(args...)
			future.Val = result
			close(future.Done)
		}()

	case *object.Closure:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		for i := range args {
			args[i] = object.EnsureResolved(args[i])
		}
		future := vm.spawnClosure(fn, args)
		vm.pop()
		vm.push(future)

	default:
		vm.callFunction(numArgs)
	}
}

// snapshotGlobals copies the parent's globals while no goroutine is racing it.
// Closures are rebound to a child VM later in newSpawnVM; this method only
// hands off a safe copy of the shared slice.
func (vm *VM) snapshotGlobals() []object.Object {
	cp := make([]object.Object, len(vm.globals))
	copy(cp, vm.globals)
	return cp
}

// newSpawnVM builds a child VM that runs a single closure in its own goroutine.
// It shares the parent's constants, but gets its own operand stack and frames
// (a VM is not safe for concurrent use), plus a snapshot of the globals. Any
// closures captured from the parent are rebound to the child so that builtins
// such as map/filter call back into the child VM instead of racing the parent.
func (vm *VM) newSpawnVM(closure *object.Closure, args []object.Object, globals []object.Object) *VM {
	child := &VM{
		constants:  vm.constants,
		globals:    make([]object.Object, len(globals)),
		stack:      make([]object.Object, StackSize),
		sp:         0,
		frames:     make([]*Frame, object.MaxCallDepth),
		frameIndex: 0,
	}
	for i, g := range globals {
		if cl, ok := g.(*object.Closure); ok {
			child.globals[i] = rebindClosure(child, cl)
		} else {
			child.globals[i] = g
		}
	}

	main := rebindClosure(child, closure)
	child.push(main)
	for _, a := range args {
		if cl, ok := a.(*object.Closure); ok {
			child.push(rebindClosure(child, cl))
		} else {
			child.push(a)
		}
	}
	child.callFunction(len(args))
	return child
}

// rebindClosure returns a copy of cl whose Executor points at vm, recursively
// rebinding any closures captured as free variables.
func rebindClosure(vm *VM, cl *object.Closure) *object.Closure {
	cp := &object.Closure{Fn: cl.Fn, Free: cl.Free, Executor: vm}
	for i, f := range cp.Free {
		if fc, ok := f.(*object.Closure); ok {
			cp.Free[i] = rebindClosure(vm, fc)
		}
	}
	return cp
}

func (vm *VM) binaryOp(op compiler.Opcode, left, right object.Object) object.Object {
	left = object.EnsureResolved(left)
	right = object.EnsureResolved(right)
	if left == nil || right == nil {
		return vm.newError("E002", "type mismatch: cannot apply operator to nil")
	}
	switch {
	case left.Type() == object.INTEGER && right.Type() == object.INTEGER:
		return vm.binaryIntOp(op, left.(*object.Integer), right.(*object.Integer))
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return vm.binaryFloatOp(op, left.(*object.Float), right.(*object.Float))
	case left.Type() == object.INTEGER && right.Type() == object.FLOAT:
		return vm.binaryFloatOp(op, &object.Float{Value: float64(left.(*object.Integer).Value)}, right.(*object.Float))
	case left.Type() == object.FLOAT && right.Type() == object.INTEGER:
		return vm.binaryFloatOp(op, left.(*object.Float), &object.Float{Value: float64(right.(*object.Integer).Value)})
	default:
		return vm.newError("E002", "type mismatch: cannot apply operator between %s and %s", left.Type(), right.Type())
	}
}

func (vm *VM) binaryIntOp(op compiler.Opcode, left, right *object.Integer) object.Object {
	l, r := left.Value, right.Value
	switch op {
	case compiler.OpAdd:
		return &object.Integer{Value: l + r}
	case compiler.OpSub:
		return &object.Integer{Value: l - r}
	case compiler.OpMul:
		return &object.Integer{Value: l * r}
	case compiler.OpDiv:
		if r == 0 {
			return vm.newError("E003", "division by zero")
		}
		return &object.Integer{Value: l / r}
	case compiler.OpMod:
		if r == 0 {
			return vm.newError("E003", "modulo by zero")
		}
		return &object.Integer{Value: l % r}
	case compiler.OpPow:
		return &object.Integer{Value: int64(math.Pow(float64(l), float64(r)))}
	}
	return &object.Error{Message: fmt.Sprintf("unknown int op %d", op)}
}

func (vm *VM) binaryFloatOp(op compiler.Opcode, left, right *object.Float) object.Object {
	l, r := left.Value, right.Value
	switch op {
	case compiler.OpAdd:
		return &object.Float{Value: l + r}
	case compiler.OpSub:
		return &object.Float{Value: l - r}
	case compiler.OpMul:
		return &object.Float{Value: l * r}
	case compiler.OpDiv:
		if r == 0 {
			return vm.newError("E003", "division by zero")
		}
		return &object.Float{Value: l / r}
	case compiler.OpMod:
		if r == 0 {
			return vm.newError("E003", "modulo by zero")
		}
		return &object.Float{Value: float64(int64(l) % int64(r))}
	case compiler.OpPow:
		return &object.Float{Value: math.Pow(l, r)}
	}
	return &object.Error{Message: fmt.Sprintf("unknown float op %d", op)}
}

// concatOp implements OpConcat: `++` concatenates strings (and bytes+string
// pairs); any other operand types are a type mismatch. This mirrors the
// tree-walker, which rejects mixed-type concat instead of silently stringifying.
func (vm *VM) concatOp() {
	right := vm.pop()
	left := vm.pop()
	right = object.EnsureResolved(right)
	left = object.EnsureResolved(left)
	if left == nil || right == nil {
		vm.push(vm.newError("E002", "type mismatch: cannot apply operator to nil"))
		return
	}
	switch {
	case left.Type() == object.STRING && right.Type() == object.STRING:
		vm.push(&object.String{Value: left.(*object.String).Value + right.(*object.String).Value})
	case left.Type() == object.BYTES && right.Type() == object.BYTES:
		l := left.(*object.Bytes).Value
		r := right.(*object.Bytes).Value
		out := make([]byte, 0, len(l)+len(r))
		out = append(out, l...)
		out = append(out, r...)
		vm.push(&object.Bytes{Value: out})
	case left.Type() == object.BYTES && right.Type() == object.STRING:
		vm.push(vm.concatBytesString(left.(*object.Bytes).Value, []byte(right.(*object.String).Value)))
	case left.Type() == object.STRING && right.Type() == object.BYTES:
		vm.push(vm.concatBytesString([]byte(left.(*object.String).Value), right.(*object.Bytes).Value))
	default:
		vm.push(vm.newError("E002", "type mismatch: cannot apply '++' between %s and %s", left.Type(), right.Type()))
	}
}

func (vm *VM) concatBytesString(l, r []byte) object.Object {
	out := make([]byte, 0, len(l)+len(r))
	out = append(out, l...)
	out = append(out, r...)
	return &object.Bytes{Value: out}
}

func (vm *VM) compareOp(op compiler.Opcode, left, right object.Object) object.Object {
	left = object.EnsureResolved(left)
	right = object.EnsureResolved(right)
	if left == nil || right == nil {
		return object.NILOBJ
	}
	switch {
	case left.Type() == object.INTEGER && right.Type() == object.INTEGER:
		return vm.compareIntOp(op, left.(*object.Integer).Value, right.(*object.Integer).Value)
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return vm.compareFloatOp(op, left.(*object.Float).Value, right.(*object.Float).Value)
	case left.Type() == object.INTEGER && right.Type() == object.FLOAT:
		return vm.compareFloatOp(op, float64(left.(*object.Integer).Value), right.(*object.Float).Value)
	case left.Type() == object.FLOAT && right.Type() == object.INTEGER:
		return vm.compareFloatOp(op, left.(*object.Float).Value, float64(right.(*object.Integer).Value))
	case left.Type() == object.BOOLEAN && right.Type() == object.BOOLEAN:
		a := object.NativeBoolToBoolean(left == object.TRUE)
		b := object.NativeBoolToBoolean(right == object.TRUE)
		return vm.compareBoolOp(op, a == object.TRUE, b == object.TRUE)
	case left.Type() == object.STRING && right.Type() == object.STRING:
		return vm.compareStringOp(op, left.(*object.String).Value, right.(*object.String).Value)
	case left.Type() == object.BYTES && right.Type() == object.BYTES:
		return vm.compareBytesOp(op, left.(*object.Bytes).Value, right.(*object.Bytes).Value)
	case op == compiler.OpEqual:
		return object.NativeBoolToBoolean(left == right)
	case op == compiler.OpNotEqual:
		return object.NativeBoolToBoolean(left != right)
	}
	return &object.Error{Message: fmt.Sprintf("Type error: comparing %s %s", left.Type(), right.Type())}
}

func (vm *VM) compareBytesOp(op compiler.Opcode, l, r []byte) object.Object {
	c := bytes.Compare(l, r)
	switch op {
	case compiler.OpEqual:
		return object.NativeBoolToBoolean(c == 0)
	case compiler.OpNotEqual:
		return object.NativeBoolToBoolean(c != 0)
	case compiler.OpGreater:
		return object.NativeBoolToBoolean(c > 0)
	case compiler.OpLess:
		return object.NativeBoolToBoolean(c < 0)
	case compiler.OpGte:
		return object.NativeBoolToBoolean(c >= 0)
	case compiler.OpLte:
		return object.NativeBoolToBoolean(c <= 0)
	}
	return &object.Error{Message: fmt.Sprintf("unknown bytes op %d", op)}
}

func (vm *VM) compareIntOp(op compiler.Opcode, l, r int64) object.Object {
	switch op {
	case compiler.OpEqual:
		return object.NativeBoolToBoolean(l == r)
	case compiler.OpNotEqual:
		return object.NativeBoolToBoolean(l != r)
	case compiler.OpGreater:
		return object.NativeBoolToBoolean(l > r)
	case compiler.OpLess:
		return object.NativeBoolToBoolean(l < r)
	case compiler.OpGte:
		return object.NativeBoolToBoolean(l >= r)
	case compiler.OpLte:
		return object.NativeBoolToBoolean(l <= r)
	}
	return object.FALSE
}

func (vm *VM) compareFloatOp(op compiler.Opcode, l, r float64) object.Object {
	switch op {
	case compiler.OpEqual:
		return object.NativeBoolToBoolean(l == r)
	case compiler.OpNotEqual:
		return object.NativeBoolToBoolean(l != r)
	case compiler.OpGreater:
		return object.NativeBoolToBoolean(l > r)
	case compiler.OpLess:
		return object.NativeBoolToBoolean(l < r)
	case compiler.OpGte:
		return object.NativeBoolToBoolean(l >= r)
	case compiler.OpLte:
		return object.NativeBoolToBoolean(l <= r)
	}
	return object.FALSE
}

func (vm *VM) compareBoolOp(op compiler.Opcode, l, r bool) object.Object {
	switch op {
	case compiler.OpEqual:
		return object.NativeBoolToBoolean(l == r)
	case compiler.OpNotEqual:
		return object.NativeBoolToBoolean(l != r)
	}
	return object.FALSE
}

func (vm *VM) compareStringOp(op compiler.Opcode, l, r string) object.Object {
	c := strings.Compare(l, r)
	switch op {
	case compiler.OpEqual:
		return object.NativeBoolToBoolean(c == 0)
	case compiler.OpNotEqual:
		return object.NativeBoolToBoolean(c != 0)
	case compiler.OpGreater:
		return object.NativeBoolToBoolean(c > 0)
	case compiler.OpLess:
		return object.NativeBoolToBoolean(c < 0)
	case compiler.OpGte:
		return object.NativeBoolToBoolean(c >= 0)
	case compiler.OpLte:
		return object.NativeBoolToBoolean(c <= 0)
	}
	return object.FALSE
}

func (vm *VM) callUserFunction(fn object.Object, args ...object.Object) object.Object {
	savedFrameIdx := vm.frameIndex
	savedSp := vm.sp

	vm.push(fn)
	for _, arg := range args {
		vm.push(arg)
	}
	vm.callFunction(len(args))

	result := vm.executeFrame()

	vm.frameIndex = savedFrameIdx
	vm.sp = savedSp
	return result
}

func (vm *VM) executeFrame() object.Object {
	// Run the frame this invocation was entered with to completion. Nested
	// user-function calls (OpCall) push additional frames and execute inline
	// in the same loop; a return from a nested frame must resume its caller
	// instead of exiting the loop entirely.
	targetFrameIdx := vm.frameIndex
	for {
		frame := vm.currentFrame()
		ins := frame.instructions

		if frame.ip >= len(ins) {
			break
		}

		op := compiler.Opcode(ins[frame.ip])
		vm.curLine = frame.lineAt(frame.ip)
		frame.ip++

		switch op {
		case compiler.OpConstant:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.constants[idx])

		case compiler.OpTrue:
			vm.push(object.TRUE)
		case compiler.OpFalse:
			vm.push(object.FALSE)
		case compiler.OpNil:
			vm.push(object.NILOBJ)

		case compiler.OpPop:
			vm.pop()

		case compiler.OpDup:
			if vm.sp > 0 {
				vm.push(vm.stack[vm.sp-1])
			}

		case compiler.OpMinus:
			val := vm.pop()
			switch v := val.(type) {
			case *object.Integer:
				vm.push(&object.Integer{Value: -v.Value})
			case *object.Float:
				vm.push(&object.Float{Value: -v.Value})
			default:
				return &object.Error{Message: fmt.Sprintf("Type error: -%s", val.Type())}
			}

		case compiler.OpNot:
			val := vm.pop()
			vm.push(object.NativeBoolToBoolean(!object.IsTruthy(val)))

		case compiler.OpAdd, compiler.OpSub, compiler.OpMul, compiler.OpDiv, compiler.OpMod, compiler.OpPow:
			right := vm.pop()
			left := vm.pop()
			result := vm.binaryOp(op, left, right)
			vm.push(result)

		case compiler.OpEqual, compiler.OpNotEqual, compiler.OpGreater, compiler.OpLess,
			compiler.OpGte, compiler.OpLte:
			right := vm.pop()
			left := vm.pop()
			result := vm.compareOp(op, left, right)
			vm.push(result)

		case compiler.OpConcat:
			vm.concatOp()

		case compiler.OpJump:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip = int(target)

		case compiler.OpJumpNotTruthy:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			if !object.IsTruthy(vm.pop()) {
				frame.ip = int(target)
			}

		case compiler.OpJumpBackward:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip = int(target)

		case compiler.OpGetGlobal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.globals[idx])

		case compiler.OpSetGlobal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.globals[idx] = vm.pop()

		case compiler.OpGetLocal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.stack[frame.basePointer+int(idx)])

		case compiler.OpSetLocal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.stack[frame.basePointer+int(idx)] = vm.pop()

		case compiler.OpGetBuiltin:
			idx := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			if idx < 0 || idx >= len(object.Builtins) {
				return &object.Error{Message: fmt.Sprintf("unknown builtin: %d", idx)}
			}
			vm.push(&object.BuiltinInfo{
				Name: object.Builtins[idx].Name,
				Fn:   object.Builtins[idx].Fn,
			})

		case compiler.OpGetFree:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(frame.closure.Free[int(idx)])

		case compiler.OpCheckError:
			val := vm.peek()
			if _, isErr := val.(*object.Error); isErr {
				vm.pendingError = nil
				vm.push(object.TRUE)
			} else if vm.pendingError != nil {
				// The error's value was discarded before the check (e.g. an
				// expression statement inside try); surface it so the catch
				// binding can convert it with OpErrorToString.
				vm.stack[vm.sp-1] = vm.pendingError
				vm.pendingError = nil
				vm.push(object.TRUE)
			} else {
				vm.push(object.FALSE)
			}

		case compiler.OpTestAbortIfError:
			vm.runTestAbortIfError(ins, frame)

		case compiler.OpTestResult:
			vm.runTestResult(ins, frame)

		case compiler.OpTryAIFix:
			src := vm.pop()
			srcStr, isStr := src.(*object.String)
			if !isStr || object.TryAIEvalFn == nil {
				vm.push(object.NILOBJ)
				continue
			}
			fixed := object.TryAIEvalFn(srcStr.Value)
			vm.push(fixed)

		case compiler.OpErrorToString:
			vm.pendingError = nil
			val := vm.pop()
			if err, isErr := val.(*object.Error); isErr {
				vm.push(&object.String{Value: err.Message})
			} else {
				vm.push(object.NILOBJ)
			}

		case compiler.OpCall:
			numArgs := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			vm.callFunction(numArgs)

		case compiler.OpSpawn:
			numArgs := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			vm.spawnCall(numArgs)

		case compiler.OpClosure:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			numFree := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			fn, ok := vm.constants[idx].(*object.CompiledFunction)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("not a CompiledFunction at index %d", idx)}
			}
			free := make([]object.Object, numFree)
			for i := numFree - 1; i >= 0; i-- {
				free[i] = vm.pop()
			}
			closure := &object.Closure{
				Fn:       fn,
				Free:     free,
				Executor: vm,
			}
			vm.push(closure)

		case compiler.OpList:
			numElems := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			elems := make([]object.Object, numElems)
			for i := numElems - 1; i >= 0; i-- {
				elems[i] = vm.pop()
			}
			vm.push(&object.List{Elements: elems})

		case compiler.OpDot:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			field := vm.constants[idx].(*object.String).Value
			obj := vm.pop()
			if obj == nil {
				vm.push(vm.newError("E006", "cannot use .%s on nil", field))
				continue
			}
			switch m := obj.(type) {
			case *object.StructInstance:
				if val, ok := m.Values[field]; ok {
					vm.push(val)
				} else {
					vm.push(object.NILOBJ)
				}
			case *object.Map:
				if val, ok := m.Pairs[field]; ok {
					vm.push(val)
				} else {
					vm.push(object.NILOBJ)
				}
			case *object.Error:
				if field == "message" {
					vm.push(&object.String{Value: m.Message})
				} else {
					vm.push(m)
				}
			default:
				vm.push(vm.newError("E006", "cannot use .%s on %s", field, obj.Type()))
			}

		case compiler.OpReturn:
			frame := vm.currentFrame()
			vm.sp = frame.savedSp
			vm.frameIndex--
			if vm.frameIndex < targetFrameIdx {
				return object.NILOBJ
			}
			vm.push(object.NILOBJ)

		case compiler.OpReturnValue:
			frame := vm.currentFrame()
			returnVal := vm.pop()
			vm.sp = frame.savedSp
			vm.frameIndex--
			if vm.frameIndex < targetFrameIdx {
				return returnVal
			}
			vm.push(returnVal)

		default:
			return &object.Error{Message: fmt.Sprintf("unknown opcode in user fn: %d", op)}
		}
	}
	return object.NILOBJ
}
