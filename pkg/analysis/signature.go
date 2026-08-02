package analysis

import (
	"github.com/harry/pipe/pkg/lexer"
)

// SignatureInfo is the LSP signature-help payload for a call site.
type SignatureInfo struct {
	Label       string
	Doc         string
	Params      []Param
	ActiveParam int // 0-based index of the parameter under the cursor, -1 if none
}

// callFrame tracks one open '(' during the scan for signature help.
type callFrame struct {
	name       string // callee name, "" for grouping parens
	commaCount int
}

// SignatureHelp returns parameter information for the call whose opening
// paren is active at the position. ok is false if no call is being edited.
func (a *Analysis) SignatureHelp(source string, line, col int) (SignatureInfo, bool) {
	toks := tokenizeAll(source)

	var stack []*callFrame
	var call *callFrame

	for i := range toks {
		t := &toks[i]
		if t.Line > line || (t.Line == line && t.Col >= col) {
			break
		}
		switch t.Type {
		case lexer.NEWLINE, lexer.INDENT, lexer.DEDENT, lexer.EOF:
			continue
		case lexer.LPAREN:
			name := ""
			if prev := prevSignificantBefore(toks, i); prev != nil && prev.Type == lexer.IDENT {
				name = prev.Literal
			}
			f := &callFrame{name: name}
			stack = append(stack, f)
			call = f
		case lexer.RPAREN:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				call = stack[len(stack)-1]
			} else {
				call = nil
			}
		case lexer.COMMA:
			if len(stack) > 0 {
				stack[len(stack)-1].commaCount++
			}
		}
	}

	if call == nil || call.name == "" {
		return SignatureInfo{}, false
	}

	// User function?
	if a != nil {
		if sym := a.SymbolAtName(call.name); sym != nil && sym.Kind == KindFunction {
			return SignatureInfo{
				Label:       call.name + "(…)",
				Doc:         sym.Doc,
				ActiveParam: call.commaCount,
			}, true
		}
	}

	// Builtin?
	if b, ok := Builtin(call.name); ok {
		return SignatureInfo{
			Label:       b.Signature,
			Doc:         b.Description,
			Params:      b.Params,
			ActiveParam: call.commaCount,
		}, true
	}

	return SignatureInfo{}, false
}

// prevSignificantBefore returns the last non-layout token before index i.
func prevSignificantBefore(toks []ctxToken, i int) *ctxToken {
	for j := i - 1; j >= 0; j-- {
		t := toks[j]
		switch t.Type {
		case lexer.NEWLINE, lexer.INDENT, lexer.DEDENT, lexer.EOF:
			continue
		}
		return &toks[j]
	}
	return nil
}
