package compiler

import (
	"github.com/MachuraHarry/pipe/pkg/ast"
	"github.com/MachuraHarry/pipe/pkg/object"
)

// literalToObject converts a literal AST node into its object value. It
// returns false for any node that is not a plain literal, so callers can
// fall back to compiling the expression for runtime evaluation.
func literalToObject(n ast.Expression) (object.Object, bool) {
	switch lit := n.(type) {
	case *ast.IntegerLiteral:
		return &object.Integer{Value: lit.Value}, true
	case *ast.FloatLiteral:
		return &object.Float{Value: lit.Value}, true
	case *ast.StringLiteral:
		return &object.String{Value: lit.Value}, true
	case *ast.BooleanLiteral:
		return object.NativeBoolToBoolean(lit.Value), true
	case *ast.NilLiteral:
		return object.NILOBJ, true
	}
	return nil, false
}

// emitConstant pushes a folded value onto the stack, reusing the compact
// encodings the compiler already uses for literal booleans and nil.
func (c *Compiler) emitConstant(obj object.Object) {
	switch v := obj.(type) {
	case *object.Boolean:
		if v.Value {
			c.emit(OpTrue)
		} else {
			c.emit(OpFalse)
		}
	case *object.NilObject:
		c.emit(OpNil)
	default:
		c.emit(OpConstant, c.addConstant(obj))
	}
}

// foldExpression tries to evaluate an expression made entirely of literals at
// compile time, mirroring the runtime semantics of the tree-walker and the VM
// exactly. It returns the folded value and true, or nil/false when any part of
// the expression depends on the runtime environment or can raise a runtime
// error, in which case the expression is compiled to ordinary bytecode.
//
// The evaluation is recursive, so a constant sub-expression nested inside a
// non-constant expression (e.g. `x + 2 * 3`) still collapses the inner part:
// compilation descends into the left/right operands, which fold independently.
func (c *Compiler) foldExpression(n ast.Expression) (object.Object, bool) {
	switch e := n.(type) {
	case *ast.PrefixExpression:
		right, ok := c.foldExpression(e.Right)
		if !ok {
			return nil, false
		}
		return foldPrefixValue(e.Operator, right)
	case *ast.InfixExpression:
		left, lok := c.foldExpression(e.Left)
		if !lok {
			return nil, false
		}
		right, rok := c.foldExpression(e.Right)
		if !rok {
			return nil, false
		}
		return foldInfixValues(e.Operator, left, right)
	}
	return literalToObject(n)
}

func foldPrefixValue(op string, right object.Object) (object.Object, bool) {
	switch op {
	case "-":
		switch v := right.(type) {
		case *object.Integer:
			return &object.Integer{Value: -v.Value}, true
		case *object.Float:
			return &object.Float{Value: -v.Value}, true
		}
	case "!":
		return object.NativeBoolToBoolean(!object.IsTruthy(right)), true
	}
	return nil, false
}

// foldInfixValues evaluates a binary operator between two folded literal
// values. It mirrors the VM's binaryOp/compareOp semantics and deliberately
// leaves operators that can raise runtime errors (/, %, **) or that depend on
// the runtime environment ([]) to the interpreter. Short-circuit operators are
// only folded here because both operands are already known to be side-effect
// free literals, so no evaluation is ever skipped that would run at runtime.
func foldInfixValues(op string, left, right object.Object) (object.Object, bool) {
	switch op {
	case "+", "-", "*":
		return foldArithmetic(op, left, right)
	case "==", "!=", "<", ">", "<=", ">=":
		return foldComparison(op, left, right)
	case "++":
		if l, ok := left.(*object.String); ok {
			if r, ok := right.(*object.String); ok {
				return &object.String{Value: l.Value + r.Value}, true
			}
		}
	case "&&":
		if !object.IsTruthy(left) {
			return left, true
		}
		return right, true
	case "||":
		if object.IsTruthy(left) {
			return left, true
		}
		return right, true
	}
	return nil, false
}

// foldArithmetic evaluates +, - and * on literal integers and floats with the
// same promotion rules as the runtime: int/int stays int (with int64
// wrapping), any float operand promotes the whole operation to float64.
func foldArithmetic(op string, left, right object.Object) (object.Object, bool) {
	switch l := left.(type) {
	case *object.Integer:
		switch r := right.(type) {
		case *object.Integer:
			return foldIntArithmetic(op, l.Value, r.Value)
		case *object.Float:
			return foldFloatArithmetic(op, float64(l.Value), r.Value)
		}
	case *object.Float:
		switch r := right.(type) {
		case *object.Float:
			return foldFloatArithmetic(op, l.Value, r.Value)
		case *object.Integer:
			return foldFloatArithmetic(op, l.Value, float64(r.Value))
		}
	}
	return nil, false
}

func foldIntArithmetic(op string, l, r int64) (object.Object, bool) {
	switch op {
	case "+":
		return &object.Integer{Value: l + r}, true
	case "-":
		return &object.Integer{Value: l - r}, true
	case "*":
		return &object.Integer{Value: l * r}, true
	}
	return nil, false
}

func foldFloatArithmetic(op string, l, r float64) (object.Object, bool) {
	switch op {
	case "+":
		return &object.Float{Value: l + r}, true
	case "-":
		return &object.Float{Value: l - r}, true
	case "*":
		return &object.Float{Value: l * r}, true
	}
	return nil, false
}

// foldComparison evaluates comparison operators between two literals with the
// same semantics as the runtime's compareOp: numeric operands (including
// mixed int/float, promoted to float64), strings by value, and booleans only
// for == and !=. Anything else is left to runtime evaluation so a potential
// type error surfaces at the same place as before.
func foldComparison(op string, left, right object.Object) (object.Object, bool) {
	switch {
	case left.Type() == object.INTEGER && right.Type() == object.INTEGER:
		return foldIntComparison(op, left.(*object.Integer).Value, right.(*object.Integer).Value), true
	case left.Type() == object.FLOAT && right.Type() == object.FLOAT:
		return foldFloatComparison(op, left.(*object.Float).Value, right.(*object.Float).Value), true
	case left.Type() == object.INTEGER && right.Type() == object.FLOAT:
		return foldFloatComparison(op, float64(left.(*object.Integer).Value), right.(*object.Float).Value), true
	case left.Type() == object.FLOAT && right.Type() == object.INTEGER:
		return foldFloatComparison(op, left.(*object.Float).Value, float64(right.(*object.Integer).Value)), true
	case left.Type() == object.STRING && right.Type() == object.STRING:
		return foldStringComparison(op, left.(*object.String).Value, right.(*object.String).Value), true
	case left.Type() == object.BOOLEAN && right.Type() == object.BOOLEAN:
		if op == "==" {
			return object.NativeBoolToBoolean(left == right), true
		}
		if op == "!=" {
			return object.NativeBoolToBoolean(left != right), true
		}
	}
	return nil, false
}

func foldIntComparison(op string, l, r int64) object.Object {
	switch op {
	case "==":
		return object.NativeBoolToBoolean(l == r)
	case "!=":
		return object.NativeBoolToBoolean(l != r)
	case "<":
		return object.NativeBoolToBoolean(l < r)
	case ">":
		return object.NativeBoolToBoolean(l > r)
	case "<=":
		return object.NativeBoolToBoolean(l <= r)
	case ">=":
		return object.NativeBoolToBoolean(l >= r)
	}
	return object.FALSE
}

func foldFloatComparison(op string, l, r float64) object.Object {
	switch op {
	case "==":
		return object.NativeBoolToBoolean(l == r)
	case "!=":
		return object.NativeBoolToBoolean(l != r)
	case "<":
		return object.NativeBoolToBoolean(l < r)
	case ">":
		return object.NativeBoolToBoolean(l > r)
	case "<=":
		return object.NativeBoolToBoolean(l <= r)
	case ">=":
		return object.NativeBoolToBoolean(l >= r)
	}
	return object.FALSE
}

func foldStringComparison(op string, l, r string) object.Object {
	switch op {
	case "==":
		return object.NativeBoolToBoolean(l == r)
	case "!=":
		return object.NativeBoolToBoolean(l != r)
	case "<":
		return object.NativeBoolToBoolean(l < r)
	case ">":
		return object.NativeBoolToBoolean(l > r)
	case "<=":
		return object.NativeBoolToBoolean(l <= r)
	case ">=":
		return object.NativeBoolToBoolean(l >= r)
	}
	return object.FALSE
}
