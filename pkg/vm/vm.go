package vm

import (
	"fmt"

	"github.com/harry/pipe/pkg/compiler"
	"github.com/harry/pipe/pkg/object"
)

const MaxFrames = 1024
const StackSize = 4096

type VM struct {
	constants    []object.Object
	globals      []object.Object
	stack        []object.Object
	sp           int
	frames       []*Frame
	frameIndex   int
	instructions compiler.Instructions
	ip           int // instruction pointer (in current frame)
}

type Frame struct {
	closure      *object.CompiledFunction
	ip           int
	basePointer  int
	instructions compiler.Instructions
}

func New(bc *compiler.Bytecode) *VM {
	stack := make([]object.Object, StackSize)
	globals := make([]object.Object, 65536)
	frames := make([]*Frame, MaxFrames)

	mainFn := &object.CompiledFunction{
		Instructions: bc.Instructions,
	}
	mainFrame := &Frame{
		closure:      mainFn,
		ip:           0,
		basePointer:  0,
		instructions: bc.Instructions,
	}

	frames[0] = mainFrame

	return &VM{
		constants:    bc.Constants,
		globals:      globals,
		stack:        stack,
		sp:           0,
		frames:       frames,
		frameIndex:   0,
	}
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

		case compiler.OpAdd, compiler.OpSub, compiler.OpMul, compiler.OpDiv, compiler.OpMod:
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
			ls, ok := left.(*object.String)
			rs, ok2 := right.(*object.String)
			if !ok || !ok2 {
				return fmt.Errorf("Typ-Fehler: ++ benötigt zwei Strings")
			}
			vm.push(&object.String{Value: ls.Value + rs.Value})

		case compiler.OpMinus:
			val := vm.pop()
			switch v := val.(type) {
			case *object.Integer:
				vm.push(&object.Integer{Value: -v.Value})
			case *object.Float:
				vm.push(&object.Float{Value: -v.Value})
			default:
				return fmt.Errorf("Typ-Fehler: -%s", val.Type())
			}

		case compiler.OpJump:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip = int(target)

		case compiler.OpJumpBackward:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip = int(target)

		case compiler.OpJumpNotTruthy:
			target := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			condition := vm.pop()
			if !object.IsTruthy(condition) {
				frame.ip = int(target)
			}

		case compiler.OpGetGlobal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			if int(idx) >= len(vm.globals) || vm.globals[idx] == nil {
				return fmt.Errorf("undefinierte Variable an Index %d", idx)
			}
			vm.push(vm.globals[idx])

		case compiler.OpSetGlobal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			v := vm.pop()
			for int(idx) >= len(vm.globals) {
				vm.globals = append(vm.globals, nil)
			}
			vm.globals[idx] = v

		case compiler.OpGetLocal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			vm.push(vm.stack[frame.basePointer+int(idx)])

		case compiler.OpSetLocal:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			v := vm.peek()
			vm.stack[frame.basePointer+int(idx)] = v

		case compiler.OpGetBuiltin:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			if int(idx) >= len(object.Builtins) {
				return fmt.Errorf("unbekannte Builtin-Funktion: %d", idx)
			}
			bi := &object.BuiltinInfo{
				Name: object.Builtins[idx].Name,
				Fn:   object.Builtins[idx].Fn,
			}
			vm.push(bi)

		case compiler.OpCall:
			numArgs := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			vm.callFunction(numArgs)

		case compiler.OpReturn:
			// Return nil
			frame := vm.currentFrame()
			vm.sp = frame.basePointer - 1
			vm.frameIndex--
			if vm.frameIndex < 0 {
				return nil
			}
			vm.push(object.NILOBJ)

		case compiler.OpReturnValue:
			frame := vm.currentFrame()
			returnVal := vm.pop()
			vm.sp = frame.basePointer - 1
			vm.frameIndex--
			if vm.frameIndex < 0 {
				return nil
			}
			vm.push(returnVal)

		case compiler.OpClosure:
			idx := compiler.ReadUint16(ins, frame.ip)
			frame.ip += 2
			numLocals := int(compiler.ReadUint16(ins, frame.ip))
			frame.ip += 2
			fn, ok := vm.constants[idx].(*object.CompiledFunction)
			if !ok {
				return fmt.Errorf("keine CompiledFunction an Index %d", idx)
			}
			fn.NumLocals = numLocals
			vm.push(fn)

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
					return fmt.Errorf("Feld '%s' nicht gefunden", field)
				}
			default:
				return fmt.Errorf("Punkt-Zugriff nur auf Maps möglich")
			}

		case compiler.OpHalt:
			return nil

		default:
			return fmt.Errorf("unbekannter Opcode: %d", op)
		}
	}

	return nil
}

func (vm *VM) callFunction(numArgs int) {
	callee := vm.stack[vm.sp-1-numArgs]

	switch fn := callee.(type) {
	case *object.CompiledFunction:
		inst, ok := fn.Instructions.(compiler.Instructions)
		if !ok {
			panic("invalid compiled function instructions")
		}

		frame := &Frame{
			closure:      fn,
			ip:           0,
			basePointer:  vm.sp - numArgs,  // point to first argument
			instructions: inst,
		}

		vm.frameIndex++
		if vm.frameIndex >= MaxFrames {
			panic("stack overflow: too many frames")
		}
		vm.frames[vm.frameIndex] = frame

	case *object.BuiltinInfo:
		args := make([]object.Object, numArgs)
		for i := numArgs - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		result := fn.Fn(args...)
		vm.pop() // pop the builtin object
		vm.push(result)

	default:
		panic(fmt.Sprintf("calling non-function: %s", callee.Type()))
	}
}

func (vm *VM) binaryOp(op compiler.Opcode, left, right object.Object) object.Object {
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
		return &object.Error{Message: fmt.Sprintf("Typ-Fehler: %s %s %s", left.Type(), op, right.Type())}
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
			return &object.Error{Message: "Division durch Null"}
		}
		return &object.Integer{Value: l / r}
	case compiler.OpMod:
		if r == 0 {
			return &object.Error{Message: "Modulo durch Null"}
		}
		return &object.Integer{Value: l % r}
	}
	return &object.Error{Message: "unbekannter Operator"}
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
			return &object.Error{Message: "Division durch Null"}
		}
		return &object.Float{Value: l / r}
	}
	return &object.Error{Message: "unbekannter Operator"}
}

func (vm *VM) compareOp(op compiler.Opcode, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER && right.Type() == object.INTEGER:
		return vm.compareIntOp(op, left.(*object.Integer), right.(*object.Integer))
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return vm.compareFloatOp(op, left.(*object.Float), right.(*object.Float))
	case left.Type() == object.STRING && right.Type() == object.STRING:
		return vm.compareStringOp(op, left.(*object.String), right.(*object.String))
	case op == compiler.OpEqual:
		return object.NativeBoolToBoolean(false)
	case op == compiler.OpNotEqual:
		return object.NativeBoolToBoolean(true)
	}
	return &object.Error{Message: fmt.Sprintf("Typ-Fehler: %s %s %s", left.Type(), op, right.Type())}
}

func (vm *VM) compareIntOp(op compiler.Opcode, left, right *object.Integer) object.Object {
	l, r := left.Value, right.Value
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

func (vm *VM) compareFloatOp(op compiler.Opcode, left, right *object.Float) object.Object {
	l, r := left.Value, right.Value
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

func (vm *VM) compareStringOp(op compiler.Opcode, left, right *object.String) object.Object {
	switch op {
	case compiler.OpEqual:
		return object.NativeBoolToBoolean(left.Value == right.Value)
	case compiler.OpNotEqual:
		return object.NativeBoolToBoolean(left.Value != right.Value)
	}
	return &object.Error{Message: fmt.Sprintf("Typ-Fehler: %s %s %s", left.Type(), op, right.Type())}
}
