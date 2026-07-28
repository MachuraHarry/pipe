package compiler

import "fmt"

type Opcode byte

const (
	OpConstant Opcode = iota
	OpTrue
	OpFalse
	OpNil
	OpPop
	OpDup
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpPow
	OpEqual
	OpNotEqual
	OpGreater
	OpLess
	OpGte
	OpLte
	OpConcat
	OpMinus
	OpNot
	OpJump
	OpJumpNotTruthy
	OpJumpBackward
	OpGetGlobal
	OpSetGlobal
	OpGetLocal
	OpSetLocal
	OpGetBuiltin
	OpCall
	OpReturn
	OpReturnValue
	OpClosure
	OpList
	OpMap
	OpDot
	OpHalt
	OpGetFree
	OpCheckError
)

var opcodeNames = map[Opcode]string{
	OpConstant:      "OpConstant",
	OpTrue:          "OpTrue",
	OpFalse:         "OpFalse",
	OpNil:           "OpNil",
	OpPop:           "OpPop",
	OpDup:           "OpDup",
	OpAdd:           "OpAdd",
	OpSub:           "OpSub",
	OpMul:           "OpMul",
	OpDiv:           "OpDiv",
	OpMod:           "OpMod",
	OpPow:           "OpPow",
	OpEqual:         "OpEqual",
	OpNotEqual:      "OpNotEqual",
	OpGreater:       "OpGreater",
	OpLess:          "OpLess",
	OpGte:           "OpGte",
	OpLte:           "OpLte",
	OpConcat:        "OpConcat",
	OpMinus:         "OpMinus",
	OpNot:           "OpNot",
	OpJump:          "OpJump",
	OpJumpNotTruthy: "OpJumpNotTruthy",
	OpJumpBackward:  "OpJumpBackward",
	OpGetGlobal:     "OpGetGlobal",
	OpSetGlobal:     "OpSetGlobal",
	OpGetLocal:      "OpGetLocal",
	OpSetLocal:      "OpSetLocal",
	OpGetBuiltin:    "OpGetBuiltin",
	OpCall:          "OpCall",
	OpReturn:        "OpReturn",
	OpReturnValue:   "OpReturnValue",
	OpClosure:       "OpClosure",
	OpList:          "OpList",
	OpMap:           "OpMap",
	OpDot:           "OpDot",
	OpHalt:          "OpHalt",
	OpGetFree:       "OpGetFree",
	OpCheckError:    "OpCheckError",
}

func (o Opcode) String() string {
	if name, ok := opcodeNames[o]; ok {
		return name
	}
	return fmt.Sprintf("Opcode(%d)", o)
}

type Instructions []byte

func Make(op Opcode, operands ...int) []byte {
	var ins []byte
	ins = append(ins, byte(op))
	for _, operand := range operands {
		ins = append(ins, encodeUint16(uint16(operand))...)
	}
	return ins
}

func encodeUint16(v uint16) []byte {
	return []byte{byte(v >> 8), byte(v)}
}

func ReadUint16(ins Instructions, offset int) uint16 {
	return uint16(ins[offset])<<8 | uint16(ins[offset+1])
}
