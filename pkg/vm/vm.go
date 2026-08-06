package vm

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/compiler"
	"github.com/MachuraHarry/pipe/pkg/object"
)

const (
	StackSize = 2048
	MaxFrames = 1024
)

type Frame struct {
	closure      *object.Closure
	ip           int
	basePointer  int
	savedSp      int
	instructions compiler.Instructions
}

func New(bc *compiler.Bytecode) *VM {
	stack := make([]object.Object, StackSize)
	globals := make([]object.Object, 65536)
	frames := make([]*Frame, MaxFrames)

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
	}

	frames[0] = mainFrame

	vm := &VM{
		constants:  bc.Constants,
		globals:    globals,
		stack:      stack,
		sp:         0,
		frames:     frames,
		frameIndex: 0,
	}

	object.SetCallUserFn(vm.callUserFunction)

	return vm
}

type VM struct {
	constants  []object.Object
	globals    []object.Object
	stack      []object.Object
	sp         int
	frames     []*Frame
	frameIndex int
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.frameIndex]
}

func (vm *VM) push(obj object.Object) {
	if vm.sp >= StackSize {
		panic("stack overflow")
	}
	vm.stack[vm.sp] = obj
	vm.sp++
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

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) Run() error {
	for {
		frame := vm.currentFrame()
		ins := frame.instructions

		if frame.ip >= len(ins) {
			break
		}

		op := compiler.Opcode(ins[frame.ip])
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
			right := vm.pop()
			left := vm.pop()
			right = object.EnsureResolved(right)
			left = object.EnsureResolved(left)
			if left.Type() == object.BYTES || right.Type() == object.BYTES {
				l := leftBytes(left)
				r := leftBytes(right)
				out := make([]byte, 0, len(l)+len(r))
				out = append(out, l...)
				out = append(out, r...)
				vm.push(&object.Bytes{Value: out})
				continue
			}
			vm.push(&object.String{Value: left.Inspect() + right.Inspect()})

		case compiler.OpMinus:
			val := vm.pop()
			switch v := val.(type) {
			case *object.Integer:
				vm.push(&object.Integer{Value: -v.Value})
			case *object.Float:
				vm.push(&object.Float{Value: -v.Value})
			default:
				return fmt.Errorf("Type error: -%s", val.Type())
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
				return fmt.Errorf("unknown builtin function: %d", idx)
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
				vm.push(object.TRUE)
			} else {
				vm.push(object.FALSE)
			}

		case compiler.OpTryAIFix:
			src := vm.pop()
			srcStr, isStr := src.(*object.String)
			if !isStr || object.TryAIEvalFn == nil {
				vm.push(object.NILOBJ)
				continue
			}
			fixed := object.TryAIEvalFn(srcStr.Value)
			vm.push(fixed)

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
				return nil
			}
			vm.push(object.NILOBJ)

		case compiler.OpReturnValue:
			frame := vm.currentFrame()
			returnVal := vm.pop()
			vm.sp = frame.savedSp
			vm.frameIndex--
			if vm.frameIndex < 0 {
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
				return fmt.Errorf("not a CompiledFunction at index %d", idx)
			}
			free := make([]object.Object, numFree)
			for i := numFree - 1; i >= 0; i-- {
				free[i] = vm.pop()
			}
			closure := &object.Closure{
				Fn:   fn,
				Free: free,
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

		case compiler.OpDot:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			field := vm.constants[idx].(*object.String).Value
			obj := vm.pop()
			switch m := obj.(type) {
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
				return fmt.Errorf(". only on map: %s", obj.Type())
			}

		case compiler.OpHalt:
			return nil

		default:
			return fmt.Errorf("unknown opcode: %d", op)
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
			panic("invalid compiled function instructions")
		}

		basePtr := vm.sp - numArgs
		savedSp := basePtr - 1

		frame := &Frame{
			closure:      fn,
			ip:           0,
			basePointer:  basePtr,
			savedSp:      savedSp,
			instructions: inst,
		}

		vm.frameIndex++
		if vm.frameIndex >= MaxFrames {
			panic("stack overflow: too many frames")
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
		for i := range args {
			args[i] = object.EnsureResolved(args[i])
		}
		result := fn.Fn(args...)
		vm.pop()
		vm.push(result)

	default:
		panic(fmt.Sprintf("calling non-function: %s", callee.Type()))
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

	default:
		vm.callFunction(numArgs)
	}
}

func (vm *VM) binaryOp(op compiler.Opcode, left, right object.Object) object.Object {
	left = object.EnsureResolved(left)
	right = object.EnsureResolved(right)
	switch {
	case left.Type() == object.INTEGER && right.Type() == object.INTEGER:
		return vm.binaryIntOp(op, left.(*object.Integer), right.(*object.Integer))
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return vm.binaryFloatOp(op, left.(*object.Float), right.(*object.Float))
	case left.Type() == object.INTEGER && right.Type() == object.FLOAT:
		return vm.binaryFloatOp(op, &object.Float{Value: float64(left.(*object.Integer).Value)}, right.(*object.Float))
	case left.Type() == object.FLOAT && right.Type() == object.INTEGER:
		return vm.binaryFloatOp(op, left.(*object.Float), &object.Float{Value: float64(right.(*object.Integer).Value)})
	case left.Type() == object.STRING && op == compiler.OpAdd:
		return &object.String{Value: left.(*object.String).Value + right.Inspect()}
	default:
		return &object.Error{Message: fmt.Sprintf("Type error: %s %s %s", left.Type(), op, right.Type())}
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
			return &object.Error{Message: "ERROR: division by zero"}
		}
		return &object.Integer{Value: l / r}
	case compiler.OpMod:
		if r == 0 {
			return &object.Error{Message: "ERROR: modulo by zero"}
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
			return &object.Error{Message: "ERROR: division by zero"}
		}
		return &object.Float{Value: l / r}
	case compiler.OpMod:
		if r == 0 {
			return &object.Error{Message: "ERROR: modulo by zero"}
		}
		return &object.Float{Value: float64(int64(l) % int64(r))}
	case compiler.OpPow:
		return &object.Float{Value: math.Pow(l, r)}
	}
	return &object.Error{Message: fmt.Sprintf("unknown float op %d", op)}
}

func leftBytes(o object.Object) []byte {
	switch v := o.(type) {
	case *object.Bytes:
		return v.Value
	case *object.String:
		return []byte(v.Value)
	}
	return []byte(o.Inspect())
}

func (vm *VM) compareOp(op compiler.Opcode, left, right object.Object) object.Object {
	left = object.EnsureResolved(left)
	right = object.EnsureResolved(right)
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
	for {
		frame := vm.currentFrame()
		ins := frame.instructions

		if frame.ip >= len(ins) {
			break
		}

		op := compiler.Opcode(ins[frame.ip])
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
			right := vm.pop()
			left := vm.pop()
			right = object.EnsureResolved(right)
			left = object.EnsureResolved(left)
			if left.Type() == object.BYTES || right.Type() == object.BYTES {
				l := leftBytes(left)
				r := leftBytes(right)
				out := make([]byte, 0, len(l)+len(r))
				out = append(out, l...)
				out = append(out, r...)
				vm.push(&object.Bytes{Value: out})
				continue
			}
			vm.push(&object.String{Value: left.Inspect() + right.Inspect()})

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
				vm.push(object.TRUE)
			} else {
				vm.push(object.FALSE)
			}

		case compiler.OpTryAIFix:
			src := vm.pop()
			srcStr, isStr := src.(*object.String)
			if !isStr || object.TryAIEvalFn == nil {
				vm.push(object.NILOBJ)
				continue
			}
			fixed := object.TryAIEvalFn(srcStr.Value)
			vm.push(fixed)

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
				Fn:   fn,
				Free: free,
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
			switch m := obj.(type) {
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
				return &object.Error{Message: fmt.Sprintf(". only on map: %s", obj.Type())}
			}

		case compiler.OpReturn:
			frame := vm.currentFrame()
			vm.sp = frame.savedSp
			vm.frameIndex--
			return object.NILOBJ

		case compiler.OpReturnValue:
			frame := vm.currentFrame()
			returnVal := vm.pop()
			vm.sp = frame.savedSp
			vm.frameIndex--
			return returnVal

		default:
			return &object.Error{Message: fmt.Sprintf("unknown opcode in user fn: %d", op)}
		}
	}
	return object.NILOBJ
}
