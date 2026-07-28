# Bytecode VM

The Pipe VM is a stack-based virtual machine that executes bytecode compiled from the AST. It provides faster execution for compute-heavy or deeply recursive code compared to the tree-walking interpreter.

## VM Architecture

```
Source Code --> Lexer --> Parser --> AST --> Compiler --> Bytecode --> VM (Execution)
```

The pipeline consists of two distinct phases:

1. **Compilation Phase**: The compiler traverses the AST and emits bytecode instructions into a `Bytecode` structure containing the instruction stream and constant pool.
2. **Execution Phase**: The VM reads the bytecode instruction-by-instruction and executes each opcode against its operand stack.

## Components

### Compiler (`pkg/compiler/`, ~876+115 lines)

The compiler transforms the AST into a flat bytecode instruction sequence. Its key sub-components are:

#### Symbol Table

The symbol table tracks variable definitions and resolves scoped access. It uses four scope types:

| Scope | Description | Opcodes Used |
|-------|-------------|-------------|
| `GlobalScope` | Top-level variables (no enclosing scope) | `OpGetGlobal`, `OpSetGlobal` |
| `LocalScope` | Variables defined inside a function body | `OpGetLocal`, `OpSetLocal` |
| `FreeScope` | Variables from an outer scope captured in a closure | `OpGetFree` |
| `BuiltinScope` | Built-in functions (print, len, etc.) | `OpGetBuiltin` |

The symbol table is chained: when entering a new scope (e.g., inside a function), a new table is created with a pointer to the outer table. Resolution walks the chain — if a symbol isn't found locally, the outer table is consulted. Variables from outer local scopes become **free variables** and are promoted into the `FreeSymbols` list for closure capture.

#### Constant Pool

Literal values (integers, floats, strings, compiled functions) are stored in a constant pool rather than being embedded directly in the instruction stream. Each constant is referenced by its 16-bit index. This keeps instructions compact and enables the `.pipec` cache format to serialize constants efficiently.

Supported constant types:
- Integer (type byte `1`)
- Float (type byte `2`)
- String (type byte `3`)
- CompiledFunction (type byte `4`)

#### Loop Patching

During compilation of `while` and `for-in` loops, jump instructions that target not-yet-compiled positions are emitted with placeholder offsets (typically `9999`). A `LoopContext` stack tracks:
- `continueTarget`: position to jump to for `continue` (the condition check)
- `breakPatches`: list of break jump positions to back-patch after the loop body

After the loop body is fully compiled, all placeholder jumps are patched to their correct positions using `patchJump(pos, target)`.

### VM (`pkg/vm/`, ~738 lines)

The VM is a stack-based interpreter with the following characteristics:

#### Stack and Frames

| Parameter | Value | Description |
|-----------|-------|-------------|
| `StackSize` | 2048 | Maximum capacity of the operand stack |
| `MaxFrames` | 1024 | Maximum call-frame depth |
| `MaxGlobals` | 65536 | Maximum number of global variables |

Each `Frame` stores:
- `closure`: the `*Closure` currently executing
- `ip`: instruction pointer (offset into the instruction stream)
- `basePointer`: base index into the stack for local variable access
- `savedSp`: saved stack pointer for returning to the caller

Local variables are accessed relative to the frame's base pointer: `stack[basePointer + idx]`.

#### Call Frames

When a function is called, the VM:
1. Creates a new `Frame` with the callee's closure and instruction stream
2. Sets the base pointer to `sp - numArgs`
3. Grows the stack if needed to accommodate local variables (`basePtr + fn.Fn.NumLocals`)
4. Pushes the new frame onto the frame stack

When a function returns (`OpReturn` / `OpReturnValue`):
1. The return value is popped from the stack
2. The stack pointer is restored to `savedSp`
3. The frame index is decremented
4. The return value is pushed onto the caller's stack

## All 47 Opcodes

### Constants & Literals

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpConstant` | 0 | 1 (constant index, 16-bit) | Push a constant from the pool onto the stack |
| `OpTrue` | 1 | 0 | Push the `true` object onto the stack |
| `OpFalse` | 2 | 0 | Push the `false` object onto the stack |
| `OpNil` | 3 | 0 | Push the `nil` object onto the stack |

### Stack Operations

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpPop` | 4 | 0 | Pop and discard the top element |
| `OpDup` | 5 | 0 | Duplicate the top element (push a copy) |

### Arithmetic

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpAdd` | 6 | 0 | Pop two values, push `left + right` |
| `OpSub` | 7 | 0 | Pop two values, push `left - right` |
| `OpMul` | 8 | 0 | Pop two values, push `left * right` |
| `OpDiv` | 9 | 0 | Pop two values, push `left / right` |
| `OpMod` | 10 | 0 | Pop two values, push `left % right` |
| `OpPow` | 11 | 0 | Pop two values, push `left ** right` |

Binary operations transparently handle mixed integer-float arithmetic by promoting integers to floats when the other operand is a float. The `OpAdd` instruction also performs string concatenation when the left operand is a string.

### Comparison

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpEqual` | 12 | 0 | Pop two values, push boolean result of `left == right` |
| `OpNotEqual` | 13 | 0 | Pop two values, push boolean result of `left != right` |
| `OpGreater` | 14 | 0 | Pop two values, push boolean result of `left > right` |
| `OpLess` | 15 | 0 | Pop two values, push boolean result of `left < right` |
| `OpGte` | 16 | 0 | Pop two values, push boolean result of `left >= right` |
| `OpLte` | 17 | 0 | Pop two values, push boolean result of `left <= right` |

Comparisons support integers, floats, strings (equality only), and booleans (equality only).

### Unary & Concat

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpMinus` | 18 | 0 | Pop one value, push its arithmetic negation (`-x`) |
| `OpNot` | 19 | 0 | Pop one value, push logical negation (`!x`) |
| `OpConcat` | 20 | 0 | Pop two strings, push `left ++ right` |

### Variables

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpGetGlobal` | 21 | 1 (index, 16-bit) | Push a global variable by index |
| `OpSetGlobal` | 22 | 1 (index, 16-bit) | Set a global variable by index (stores `peek()`) |
| `OpGetLocal` | 23 | 1 (index, 16-bit) | Push a local variable: `stack[basePointer + idx]` |
| `OpSetLocal` | 24 | 1 (index, 16-bit) | Set a local variable: `stack[basePointer + idx] = peek()` |
| `OpGetBuiltin` | 25 | 1 (index, 16-bit) | Push a builtin function wrapper by index |
| `OpGetFree` | 26 | 1 (index, 16-bit) | Push a free variable from the closure's captured values |

### Control Flow

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpJump` | 27 | 1 (target, 16-bit) | Unconditional jump: set `ip = target` |
| `OpJumpNotTruthy` | 28 | 1 (target, 16-bit) | Pop a value; if not truthy, set `ip = target` |
| `OpJumpBackward` | 29 | 1 (target, 16-bit) | Unconditional backward jump (for loops) |
| `OpCheckError` | 30 | 0 | Push `true` if stack top is an `Error` object, else `false` |

### Functions

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpCall` | 31 | 1 (arg count, 16-bit) | Call a function with N arguments on the stack |
| `OpReturn` | 32 | 0 | Return from a function, pushing `nil` |
| `OpReturnValue` | 33 | 0 | Return from a function, pushing the return value |
| `OpClosure` | 34 | 2 (const index, free count; both 16-bit) | Create a closure from a compiled function constant |
| `OpHalt` | 35 | 0 | Halt VM execution immediately |

### Data Structures

| Opcode | Index | Operands | Description |
|--------|-------|----------|-------------|
| `OpList` | 36 | 1 (elem count, 16-bit) | Pop N elements, push a `List` object |
| `OpMap` | 37 | 1+ (pair count + key indices) | Pop N values, push a `Map` with string keys from constants |
| `OpDot` | 38 | 1 (key index, 16-bit) | Pop an object, push `obj.field` (lookup in constant pool) |

## Instruction Encoding

Each instruction is encoded as:

```
+--------+----------------+----------------+
| Opcode | Operand 1      | Operand 2      |
| 1 byte | 2 bytes (BE)   | 2 bytes (BE)   |
+--------+----------------+----------------+
```

- The opcode is always 1 byte.
- Operands are serialized as 16-bit big-endian unsigned integers.
- Instructions with no operands are 1 byte.
- Instructions with 1 operand are 3 bytes (e.g., `OpConstant`, `OpGetGlobal`).
- Instructions with 2 operands are 5 bytes (e.g., `OpClosure`).

The `Make(op, operands...)` function encodes an instruction, and `ReadUint16(ins, offset)` decodes a 16-bit operand at a given offset within the instruction stream.

## Compilation Examples

### Example 1: `1 + 2`

**AST**:
```
InfixExpression {
    Operator: "+"
    Left:  IntegerLiteral(1)
    Right: IntegerLiteral(2)
}
```

**Compilation Steps**:
1. Compile `IntegerLiteral(1)` → add `1` to constant pool at index 0 → emit `OpConstant 0`
2. Compile `IntegerLiteral(2)` → add `2` to constant pool at index 1 → emit `OpConstant 1`
3. Emit `OpAdd`

**Bytecode**:
```
0000 OpConstant     0     ; push constant[0] = 1
0003 OpConstant     1     ; push constant[1] = 2
0006 OpAdd                 ; pop 1, pop 2, push 3
```

**Constant Pool**: `[0: 1, 1: 2]`

### Example 2: `fn double x: x * 2`

**AST**:
```
FnStatement {
    Name: "double"
    Parameters: [x]
    Body: InfixExpression { Operator: "*", Left: Identifier("x"), Right: IntegerLiteral(2) }
}
```

**Compilation Steps**:
1. Define `double` (global scope, index 0)
2. Enter new scope
3. Define `x` as local (local scope, index 0)
4. Compile body:
   - `Identifier("x")` → resolve as local index 0 → emit `OpGetLocal 0`
   - `IntegerLiteral(2)` → add to constant pool → emit `OpConstant 0`
   - `*` → emit `OpMul`
5. Auto-append `OpReturnValue` (not already a return)
6. Leave scope → capture `CompiledScope{Instructions, NumLocals: 1, FreeSymbols: []}`
7. Add `CompiledFunction` to constant pool at index 0
8. Emit `OpClosure 0 0` (constant index 0, 0 free vars)
9. Emit `OpSetGlobal 0` (store in global "double")

**Bytecode**:
```
0000 OpClosure       0 0   ; create closure from constant[0]
0005 OpSetGlobal     0     ; store in globals[0] ("double")
```

**Inner function bytecode** (stored in constant[0]):
```
0000 OpGetLocal      0     ; push local[0] = x
0003 OpConstant     0     ; push constant[0] = 2
0006 OpMul                 ; multiply
0007 OpReturnValue         ; return result
```

### Example 3: Closure (`make_adder`)

```pipe
make_adder: fn base
  fn x
    base + x

add5: make_adder 5
```

**Compilation**:

1. `make_adder` enters scope; `base` is local 0
2. Inner `fn x` enters new enclosed scope:
   - `x` is local 0
   - `base` is resolved from outer scope → **FreeScope** (index 0 in `FreeSymbols`)
3. Inner body: `OpGetFree 0` + `OpGetLocal 0` + `OpAdd` + `OpReturnValue`
4. Leave inner scope → `CompiledScope{Instructions, NumLocals: 1, FreeSymbols: [{name:"base", Scope:FreeScope, Index:0}]}`
5. Outer `make_adder`: emit `OpGetLocal 0` (load `base` for free capture), then `OpClosure 0 1` (1 free var)
6. Call `make_adder 5` → closure is created capturing `base=5`

## Simplified Execute Loop Pseudocode

```go
func (vm *VM) Run() error {
    for {
        frame = vm.currentFrame()
        if frame.ip >= len(frame.instructions) { break }

        op = frame.instructions[frame.ip]
        frame.ip++

        switch op {
        case OpConstant:
            idx = readUint16()
            frame.ip += 2
            vm.push(vm.constants[idx])

        case OpAdd..OpPow:
            right = vm.pop()
            left  = vm.pop()
            vm.push(binaryOp(op, left, right))

        case OpJump:
            target = readUint16()
            frame.ip = target

        case OpJumpNotTruthy:
            target = readUint16()
            frame.ip += 2
            if !isTruthy(vm.pop()) { frame.ip = target }

        case OpCall:
            numArgs = readUint16()
            frame.ip += 2
            vm.callFunction(numArgs)

        case OpClosure:
            constIdx = readUint16()
            frame.ip += 2
            numFree = readUint16()
            frame.ip += 2
            // pop free vars from stack, create closure, push closure

        // ... (remaining opcodes)

        case OpHalt:
            return nil
        }
    }
    return nil
}
```

## Symbol Table Details

```
┌─────────────────────────────────────────────────┐
│              SymbolTable                        │
│  store: map[string]Symbol                       │
│  numDefinitions: int                            │
│  Outer: *SymbolTable  (nil for global scope)    │
│  FreeSymbols: []Symbol                          │
└─────────────────────────────────────────────────┘
```

### Resolution Rules

```
Define(name):
    if Outer == nil → GlobalScope
    else           → LocalScope

Resolve(name):
    if found in store → return symbol
    else if Outer != nil:
        outerSym = Outer.Resolve(name)
        if outerSym.Scope is GlobalScope or BuiltinScope:
            return outerSym  (direct reference)
        else:
            // outerSym is local to an enclosing function
            // → promote to free variable
            free = Symbol{name, FreeScope, len(FreeSymbols)}
            store[name] = free
            FreeSymbols = append(FreeSymbols, outerSym)
            return free
    else:
        not found → error
```

This mechanism ensures that closures correctly capture variables from enclosing scopes. Each closure stores its captured free variables in the `Free` slice, and `OpGetFree` loads them by index at runtime.

## Stack Limits

| Parameter | Value | Description |
|-----------|-------|-------------|
| Stack capacity | 2048 slots | Maximum values on the operand stack |
| Max frame depth | 1024 | Maximum nested function call depth |
| Max globals | 65536 | Maximum global variable definitions |
| Constant pool | 65536 entries | Maximum size (limited by 16-bit index) |
| Bytecode size | 65536 bytes | Maximum instruction stream size (16-bit jump targets) |

Exceeding stack capacity triggers a runtime panic (`"stack overflow"`). Exceeding max frames triggers `"stack overflow: too many frames"`.
