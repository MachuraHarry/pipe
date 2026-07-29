package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/harry/pipe/pkg/ast"
	"github.com/harry/pipe/pkg/lexer"
)

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

const (
	_ int = iota
	PrecedenceLowest
	PrecedenceOr
	PrecedenceAnd
	PrecedencePipeline
	PrecedenceEquals
	PrecedenceCompare
	PrecedenceSum
	PrecedenceProduct
	PrecedencePower
	PrecedenceConcat
	PrecedencePrefix
	PrecedenceCall
	PrecedenceDot
)

var precedences = map[lexer.TokenType]int{
	lexer.OR:       PrecedenceOr,
	lexer.AND:      PrecedenceAnd,
	lexer.ARROW:    PrecedencePipeline,
	lexer.ARROW2:   PrecedencePipeline,
	lexer.EQ:       PrecedenceEquals,
	lexer.NOT_EQ:   PrecedenceEquals,
	lexer.LT:       PrecedenceCompare,
	lexer.GT:       PrecedenceCompare,
	lexer.LTE:      PrecedenceCompare,
	lexer.GTE:      PrecedenceCompare,
	lexer.PLUS:     PrecedenceSum,
	lexer.MINUS:    PrecedenceSum,
	lexer.STAR:     PrecedenceProduct,
	lexer.SLASH:    PrecedenceProduct,
	lexer.PERCENT:  PrecedenceProduct,
	lexer.POWER:    PrecedencePower,
	lexer.CONCAT:   PrecedenceConcat,
	lexer.DOT:      PrecedenceDot,
	lexer.LBRACKET: PrecedenceCall,
}

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
	blockDepth     int // tracks nesting of expression blocks
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = map[lexer.TokenType]prefixParseFn{
		lexer.IDENT:    p.parseIdentifier,
		lexer.INT:      p.parseIntegerLiteral,
		lexer.FLOAT:    p.parseFloatLiteral,
		lexer.STRING:   p.parseStringLiteral,
		lexer.TRUE:     p.parseBooleanLiteral,
		lexer.FALSE:    p.parseBooleanLiteral,
		lexer.NIL:      p.parseNilLiteral,
		lexer.MINUS:    p.parsePrefixExpression,
		lexer.BANG:     p.parsePrefixExpression,
		lexer.LPAREN:   p.parseGroupedExpression,
		lexer.LBRACKET: p.parseListLiteral,
		lexer.LBRACE:   p.parseMapLiteral,
		lexer.IF:       p.parseIfExpression,
		lexer.MATCHKW:  p.parseMatchExpression,
		lexer.WHILE:    p.parseWhileExpression,
		lexer.FOR:      p.parseForExpression,
		lexer.FN:       p.parseFnLiteral,
		lexer.TRY:      p.parseTryExpression,
	}

	p.infixParseFns = map[lexer.TokenType]infixParseFn{
		lexer.PLUS:     p.parseInfixExpression,
		lexer.MINUS:    p.parseInfixExpression,
		lexer.STAR:     p.parseInfixExpression,
		lexer.SLASH:    p.parseInfixExpression,
		lexer.PERCENT:  p.parseInfixExpression,
		lexer.POWER:    p.parseInfixExpression,
		lexer.EQ:       p.parseInfixExpression,
		lexer.NOT_EQ:   p.parseInfixExpression,
		lexer.LT:       p.parseInfixExpression,
		lexer.ARROW:    p.parsePipelineExpression,
		lexer.ARROW2:   p.parseParallelPipelineExpression,
		lexer.GT:       p.parseInfixExpression,
		lexer.LTE:      p.parseInfixExpression,
		lexer.GTE:      p.parseInfixExpression,
		lexer.CONCAT:   p.parseInfixExpression,
		lexer.AND:      p.parseInfixExpression,
		lexer.OR:       p.parseInfixExpression,
		lexer.DOT:      p.parseDotExpression,
		lexer.LPAREN:   p.parseCallExpression,
		lexer.LBRACKET: p.parseIndexOrSlice,
	}

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) error(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d col %d: %s", p.curToken.Line, p.curToken.Col, msg))
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	p.errors = append(p.errors, fmt.Sprintf(
		"line %d col %d: expected %s, got %s (%q)",
		p.peekToken.Line, p.peekToken.Col,
		lexer.TokenName(t), lexer.TokenName(p.peekToken.Type), p.peekToken.Literal,
	))
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	p.errors = append(p.errors,
		fmt.Sprintf("line %d col %d: no prefix parser for %s (%q)",
			p.curToken.Line, p.curToken.Col, lexer.TokenName(t), p.curToken.Literal))
}

// ---- Entry points ----

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.FN:
		return p.parseFnStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.BREAK:
		return &ast.BreakStatement{}
	case lexer.CONTINUE:
		return &ast.ContinueStatement{}
	case lexer.IMPORT:
		return p.parseImportStatement()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.DEFER:
		return p.parseDeferStatement()
	case lexer.EXPORT:
		return p.parseExportStatement()
	case lexer.ENUM:
		return p.parseEnumStatement()
	case lexer.NEWLINE:
		return nil
	case lexer.DEDENT:
		return nil
	case lexer.INDENT:
		return nil
	default:
		return p.parseExpressionOrVarStatement()
	}
}

func (p *Parser) parseExpressionOrVarStatement() ast.Statement {
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
		return p.parseVarStatement()
	}
	// Compound assignment: x += expr
	if p.curTokenIs(lexer.IDENT) && isCompoundAssign(p.peekToken.Type) {
		return p.parseCompoundAssign()
	}

	stmt := &ast.ExpressionStatement{}
	stmt.Expression = p.parseExpression(PrecedenceLowest)

	p.tryPipelineContinuation(&stmt.Expression)

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) tryPipelineContinuation(expr *ast.Expression) {
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
		if p.peekTokenIs(lexer.INDENT) {
			p.nextToken()
			if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ARROW2) {
				for p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ARROW2) || p.peekTokenIs(lexer.PIPE) {
					if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ARROW2) {
						parallel := p.peekTokenIs(lexer.ARROW2)
						p.nextToken()
						p.nextToken()
						right := p.parseExpression(PrecedenceCall)
						*expr = &ast.PipelineExpression{
							Left:     *expr,
							Right:    p.insertPipelinePlaceholder(right, *expr),
							Parallel: parallel,
						}
					} else if p.peekTokenIs(lexer.PIPE) {
						p.nextToken()
						p.nextToken()
					}
					if p.peekTokenIs(lexer.NEWLINE) {
						p.nextToken()
					}
				}
				if p.peekTokenIs(lexer.DEDENT) {
					p.nextToken()
				}
			}
		}
	}
}

func isCompoundAssign(t lexer.TokenType) bool {
	switch t {
	case lexer.PLUSEQ, lexer.MINUSEQ, lexer.STAREQ, lexer.SLASHEQ, lexer.PERCENTEQ:
		return true
	}
	return false
}

func (p *Parser) parseCompoundAssign() ast.Statement {
	stmt := &ast.VarStatement{}
	stmt.Name = &ast.Identifier{Value: p.curToken.Literal}

	opToken := p.peekToken
	op := ""
	switch opToken.Type {
	case lexer.PLUSEQ:
		op = "+"
	case lexer.MINUSEQ:
		op = "-"
	case lexer.STAREQ:
		op = "*"
	case lexer.SLASHEQ:
		op = "/"
	case lexer.PERCENTEQ:
		op = "%"
	}

	p.nextToken() // skip to +=
	p.nextToken() // skip to value

	rightExpr := p.parseExpression(PrecedenceLowest)
	stmt.Value = &ast.InfixExpression{
		Operator: op,
		Left:     stmt.Name,
		Right:    rightExpr,
	}
	return stmt
}

func (p *Parser) parseVarStatement() *ast.VarStatement {
	stmt := &ast.VarStatement{}
	stmt.Name = &ast.Identifier{Value: p.curToken.Literal}

	p.nextToken()
	p.nextToken()

	stmt.Value = p.parseExpression(PrecedenceLowest)

	p.tryPipelineContinuation(&stmt.Value)

	return stmt
}

func (p *Parser) parseFnStatement() *ast.FnStatement {
	stmt := &ast.FnStatement{}

	p.nextToken()

	if !p.curTokenIs(lexer.IDENT) {
		p.error("expected function name after 'fn'")
		return nil
	}
	stmt.Name = &ast.Identifier{Value: p.curToken.Literal}

	p.nextToken()
	stmt.Parameters = p.parseParameters()

	stmt.Body = p.parseBlock()

	return stmt
}

func (p *Parser) parseParameters() []*ast.Identifier {
	var params []*ast.Identifier

	for p.curTokenIs(lexer.IDENT) && !p.peekTokenIs(lexer.COLON) {
		params = append(params, &ast.Identifier{Value: p.curToken.Literal})
		p.nextToken()
	}

	return params
}

func (p *Parser) parseBlock() *ast.BlockStatement {
	block := &ast.BlockStatement{}
	block.Statements = []ast.Statement{}

	if p.curTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	if !p.curTokenIs(lexer.INDENT) {
		expr := p.parseExpression(PrecedenceLowest)
		if expr != nil {
			block.Statements = append(block.Statements, &ast.ExpressionStatement{Expression: expr})
		}
		return block
	}

	p.nextToken()

	for !p.curTokenIs(lexer.DEDENT) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	// Consume nested DEDENTs from inner blocks (if/while inside this block)
	for p.blockDepth > 0 && p.curTokenIs(lexer.DEDENT) {
		p.blockDepth--
		p.nextToken()
	}

	return block
}

func (p *Parser) closeBlock() {
	// Currently a no-op — DEDENT handling for nested blocks is WIP
}

// ---- Expression parsing ----

func (p *Parser) parseExpression(precedence int) ast.Expression {
	return p.parseExpr(precedence, true)
}

func (p *Parser) parseExpressionNoSpace(precedence int) ast.Expression {
	return p.parseExpr(precedence, false)
}

func (p *Parser) parseExpr(precedence int, allowSpaceCalls bool) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	// FnLiteral: don't continue parsing (it's self-contained with its own block)
	if _, ok := leftExp.(*ast.FnLiteral); ok {
		return leftExp
	}

	// TryExpression: don't continue (self-contained)
	if _, ok := leftExp.(*ast.TryExpression); ok {
		return leftExp
	}

	// IfExpression, WhileExpression: don't continue (self-contained blocks)
	if _, ok := leftExp.(*ast.IfExpression); ok {
		return leftExp
	}
	if _, ok := leftExp.(*ast.WhileExpression); ok {
		return leftExp
	}

	for {
		if allowSpaceCalls &&
			!p.peekTokenIs(lexer.NEWLINE) &&
			!p.peekTokenIs(lexer.DEDENT) &&
			!p.peekTokenIs(lexer.EOF) &&
			!p.peekTokenIs(lexer.INDENT) &&
			isValueToken(p.peekToken.Type) {
			call := &ast.CallExpression{Function: leftExp}

			for !p.peekTokenIs(lexer.NEWLINE) &&
				!p.peekTokenIs(lexer.DEDENT) &&
				!p.peekTokenIs(lexer.EOF) &&
				!p.peekTokenIs(lexer.INDENT) &&
				isValueToken(p.peekToken.Type) {
				p.nextToken()
				arg := p.parseExpressionNoSpace(PrecedenceSum)
				call.Arguments = append(call.Arguments, arg)
			}

			leftExp = call
			continue
		}

		if !p.peekTokenIs(lexer.NEWLINE) &&
			!p.peekTokenIs(lexer.DEDENT) &&
			!p.peekTokenIs(lexer.EOF) &&
			!p.peekTokenIs(lexer.INDENT) &&
			precedence < p.peekPrecedence() {
			infix := p.infixParseFns[p.peekToken.Type]
			if infix != nil {
				p.nextToken()
				leftExp = infix(leftExp)
				continue
			}
		}

		break
	}

	return leftExp
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return PrecedenceLowest
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return PrecedenceLowest
}

func isValueToken(t lexer.TokenType) bool {
	switch t {
	case lexer.IDENT, lexer.INT, lexer.FLOAT, lexer.STRING,
		lexer.TRUE, lexer.FALSE, lexer.NIL,
		lexer.LPAREN, lexer.LBRACE:
		return true
	}
	return false
}

// ---- Prefix parsers ----

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{}
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.error(fmt.Sprintf("invalid number: %s", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.error(fmt.Sprintf("invalid number: %s", p.curToken.Literal))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Value: p.curTokenIs(lexer.TRUE)}
}

func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expr := &ast.PrefixExpression{Operator: p.curToken.Literal}
	p.nextToken()
	expr.Right = p.parseExpression(PrecedencePrefix)
	return expr
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken() // skip (
	exp := p.parseExpression(PrecedenceLowest)
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{}

	p.nextToken() // skip 'if'
	expr.Condition = p.parseExpression(PrecedenceLowest)

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()

	expr.Consequence = p.parseBlock()
	p.closeBlock() // consume nested block's DEDENT

	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()
		p.nextToken()
		expr.Alternative = p.parseBlock()
		p.closeBlock() // consume nested block's DEDENT
	}

	return expr
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expr := &ast.MatchExpression{}

	p.nextToken() // skip 'match'
	expr.Value = p.parseExpression(PrecedenceLowest)

	if !p.expectPeek(lexer.NEWLINE) {
		return nil
	}
	if !p.expectPeek(lexer.INDENT) {
		return nil
	}
	p.nextToken() // skip INDENT

	for p.curTokenIs(lexer.PIPE) {
		c := ast.MatchCase{}

		p.nextToken() // skip |
		c.Pattern = p.parseExpression(PrecedenceLowest)

		if !p.expectPeek(lexer.MATCH) { // ->
			return nil
		}
		p.nextToken() // skip ->
		c.Body = p.parseExpression(PrecedenceLowest)

		expr.Cases = append(expr.Cases, c)

		for p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
		if p.peekTokenIs(lexer.DEDENT) {
			p.nextToken()
			break
		}
		p.nextToken()
	}

	return expr
}

func (p *Parser) parseListLiteral() ast.Expression {
	list := &ast.ListLiteral{}
	list.Elements = p.parseExpressionList(lexer.RBRACKET)
	return list
}

func (p *Parser) parseMapLiteral() ast.Expression {
	m := &ast.MapLiteral{Pairs: make(map[string]ast.Expression)}

	if p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()
		return m
	}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.error("expected key in map")
			return nil
		}
		key := p.curToken.Literal

		if !p.expectPeek(lexer.COLON) {
			return nil
		}

		p.nextToken()
		val := p.parseExpression(PrecedenceLowest)
		m.Pairs[key] = val

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
		} else if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			break
		} else {
			p.nextToken()
		}
	}

	return m
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(PrecedenceLowest))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(PrecedenceLowest))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// ---- Infix parsers ----

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parsePipelineExpression(left ast.Expression) ast.Expression {
	if isSimpleLiteral(p.peekToken.Type) {
		return p.parseInfixExpression(left)
	}

	expr := &ast.PipelineExpression{
		Left: left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)
	expr.Right = p.insertPipelinePlaceholder(right, left)

	return expr
}

func (p *Parser) parseParallelPipelineExpression(left ast.Expression) ast.Expression {
	if isSimpleLiteral(p.peekToken.Type) {
		return p.parseInfixExpression(left)
	}

	expr := &ast.PipelineExpression{
		Left:     left,
		Parallel: true,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)
	expr.Right = p.insertPipelinePlaceholder(right, left)

	return expr
}

func (p *Parser) insertPipelinePlaceholder(right ast.Expression, pipedValue ast.Expression) ast.Expression {
	callExpr, ok := right.(*ast.CallExpression)
	if !ok {
		return right
	}
	for i, arg := range callExpr.Arguments {
		if ident, ok := arg.(*ast.Identifier); ok && ident.Value == "_" {
			callExpr.Arguments[i] = pipedValue
			callExpr.PipedArg = true
			return callExpr
		}
	}
	return right
}

func isSimpleLiteral(t lexer.TokenType) bool {
	switch t {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE, lexer.NIL, lexer.IDENT, lexer.LPAREN:
		return true
	}
	return false
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	p.nextToken()
	if !p.curTokenIs(lexer.IDENT) {
		p.error("expected field name after '.'")
		return nil
	}
	return &ast.DotExpression{
		Left:  left,
		Field: p.curToken.Literal,
	}
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	call := &ast.CallExpression{Function: function}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return call
	}

	p.nextToken()
	call.Arguments = append(call.Arguments, p.parseExpression(PrecedenceLowest))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		call.Arguments = append(call.Arguments, p.parseExpression(PrecedenceLowest))
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
	}

	return call
}

// ---- while/for/break/continue/import ----

func (p *Parser) parseWhileStatement() ast.Statement {
	expr := p.parseWhileExpression()
	return &ast.ExpressionStatement{Expression: expr}
}

func (p *Parser) parseForStatement() ast.Statement {
	expr := p.parseForExpression()
	return &ast.ExpressionStatement{Expression: expr}
}

func (p *Parser) parseWhileExpression() ast.Expression {
	expr := &ast.WhileExpression{}
	p.nextToken() // skip 'while'
	expr.Condition = p.parseExpression(PrecedenceLowest)

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.Body = p.parseBlock()
	p.closeBlock()
	return expr
}

func (p *Parser) parseForExpression() ast.Expression {
	expr := &ast.ForExpression{}
	p.nextToken() // skip 'for'

	// Check for for-in: for IDENT in expr
	if p.curTokenIs(lexer.IDENT) {
		// Save the iterator name
		iterName := p.curToken.Literal
		if p.peekTokenIs(lexer.IDENT) {
			// Check if the next token is 'in' (which is an IDENT)
			p.nextToken()
			if p.curToken.Literal == "in" {
				// for-in loop
				expr.IsForIn = true
				expr.Iterator = &ast.Identifier{Value: iterName}
				p.nextToken() // move to iterable expression
				expr.Iterable = p.parseExpression(PrecedenceLowest)

				if p.peekTokenIs(lexer.NEWLINE) {
					p.nextToken()
				}
				p.nextToken()
				expr.Body = p.parseBlock()
				p.closeBlock()
				return expr
			}
		}
	}

	// Simple for: just parse body
	if p.peekTokenIs(lexer.NEWLINE) {
		return nil
	}
	p.nextToken()

	expr.Body = p.parseBlock()
	p.closeBlock()
	return expr
}

func (p *Parser) parseReturnStatement() ast.Statement {
	p.nextToken() // skip 'return'
	return &ast.ReturnStatement{Value: p.parseExpression(PrecedenceLowest)}
}

func (p *Parser) parseDeferStatement() ast.Statement {
	p.nextToken() // skip 'defer'
	return &ast.DeferStatement{Expression: p.parseExpression(PrecedenceLowest)}
}

func (p *Parser) parseExportStatement() ast.Statement {
	p.nextToken() // skip 'export'
	if !p.curTokenIs(lexer.FN) {
		p.error("export expects 'fn'")
		return nil
	}
	fnStmt := p.parseFnStatement()
	if fnStmt != nil {
		return &ast.ExportStatement{FnName: fnStmt.Name.Value, Fn: fnStmt}
	}
	return nil
}

func (p *Parser) parseEnumStatement() ast.Statement {
	stmt := &ast.EnumStatement{}

	p.nextToken() // skip 'enum'
	if !p.curTokenIs(lexer.IDENT) {
		p.error("enum expects a name")
		return nil
	}
	stmt.Name = p.curToken.Literal

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // skip colon

	for p.curTokenIs(lexer.IDENT) {
		stmt.Values = append(stmt.Values, p.curToken.Literal)
		p.nextToken()
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // skip comma
		}
	}

	return stmt
}

func (p *Parser) parseImportStatement() ast.Statement {
	p.nextToken() // skip 'import'
	if !p.curTokenIs(lexer.STRING) {
		p.error("import expects a string path")
		return nil
	}
	stmt := &ast.ImportStatement{Path: p.curToken.Literal}

	// Optional: as <alias>
	if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "as" {
		p.nextToken() // skip 'as'
		p.nextToken() // move to alias name
		if p.curTokenIs(lexer.IDENT) {
			stmt.Alias = p.curToken.Literal
		}
	}

	return stmt
}

func (p *Parser) parseFnLiteral() ast.Expression {
	lit := &ast.FnLiteral{}

	p.nextToken() // skip 'fn'

	// Parameters
	lit.Parameters = p.parseParameters()

	// Body
	lit.Body = p.parseBlock()
	p.closeBlock()
	return lit
}

func (p *Parser) parseIndexOrSlice(left ast.Expression) ast.Expression {
	p.nextToken() // skip [

	// Empty index? []
	if p.curTokenIs(lexer.RBRACKET) {
		return left
	}

	startExp := p.parseExpression(PrecedenceLowest)

	// Slice: start..end or ..end
	if p.peekTokenIs(lexer.DOTDOT) {
		p.nextToken() // skip ..
		p.nextToken() // move to end expression
		var endExp ast.Expression
		if !p.curTokenIs(lexer.RBRACKET) {
			endExp = p.parseExpression(PrecedenceLowest)
		}
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		return &ast.SliceExpression{List: left, Start: startExp, End: endExp}
	}

	// Simple index: [expr]
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	// Return a call-like expression for index access (treated as at in evaluator)
	return &ast.InfixExpression{Operator: "[]", Left: left, Right: startExp}
}

func (p *Parser) parseTryExpression() ast.Expression {
	expr := &ast.TryExpression{}

	p.nextToken() // skip 'try'
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.TryBlock = p.parseBlock()
	p.closeBlock()

	// Expect 'catch' keyword
	if !p.peekTokenIs(lexer.CATCH) {
		p.error("try without catch")
		return nil
	}
	p.nextToken() // consume 'catch'
	p.nextToken() // move to catch param name
	if p.curTokenIs(lexer.IDENT) {
		expr.CatchParam = &ast.Identifier{Value: p.curToken.Literal}
	}
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	p.nextToken()
	expr.CatchBlock = p.parseBlock()
	p.closeBlock()

	return expr
}

func QuoteString(s string) string {
	var buf strings.Builder
	buf.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '\n':
			buf.WriteString("\\n")
		case '\t':
			buf.WriteString("\\t")
		case '\r':
			buf.WriteString("\\r")
		case '\\':
			buf.WriteString("\\\\")
		case '"':
			buf.WriteString("\\\"")
		default:
			buf.WriteRune(ch)
		}
	}
	buf.WriteByte('"')
	return buf.String()
}
