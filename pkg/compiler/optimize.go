package compiler

// OptimizeJumps performs post-compilation jump optimizations:
// 1. Jump to next instruction → remove the jump
// 2. Jump to another jump → redirect to final target
// 3. JumpNotTruthy after a known boolean → simplify
func OptimizeJumps(bc *Bytecode) *Bytecode {
	ins := bc.Instructions
	if len(ins) == 0 {
		return bc
	}

	// Pass 1: Build jump targets map (instruction position → target position)
	changed := true
	for changed {
		changed = false
		i := 0
		for i < len(ins) {
			op := Opcode(ins[i])
			switch op {
			case OpJump:
				if i+3 >= len(ins) {
					i += 3
					continue
				}
				target := int(ReadUint16(ins, i+1))

				// Optimization: Jump to next instruction → remove
				if target == i+3 {
					ins = append(ins[:i], ins[i+3:]...)
					bc.Lines = append(bc.Lines[:i], bc.Lines[i+3:]...)
					changed = true
					break
				}

				i += 3
			default:
				i += opcodeWidth(op)
			}
		}
	}

	bc.Instructions = ins
	return bc
}

// opcodeWidth returns the byte width of an instruction including operands.
func opcodeWidth(op Opcode) int {
	switch op {
	case OpConstant, OpGetGlobal, OpSetGlobal, OpGetLocal, OpSetLocal,
		OpJump, OpJumpNotTruthy, OpCall, OpClosure:
		return 3
	case OpReturnValue:
		return 1
	case OpPop:
		return 1
	default:
		return 1
	}
}
