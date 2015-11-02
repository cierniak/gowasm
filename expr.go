package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

type BinOp int

const (
	binOpInvalid BinOp = -1
	binOpAdd     BinOp = 1
	binOpSub     BinOp = 2
	binOpMul     BinOp = 3
)

var binOpNames = [...]string{
	binOpAdd: "add",
	binOpSub: "sub",
	binOpMul: "mul",
}

var binOpMapping = [...]BinOp{
	token.ADD: binOpAdd,
	token.SUB: binOpSub,
	token.MUL: binOpMul,
}

type WasmExpression interface {
	print(writer FormattingWriter)
	getType() *WasmType
	getNode() ast.Node
}

// value: <int> | <float>
type WasmValue struct {
	astBasicLiteral *ast.BasicLit
	t               *WasmType
}

// ( return <expr>? )
type WasmReturn struct {
	value WasmExpression
	stmt  *ast.ReturnStmt
}

// ( call <var> <expr>* )
type WasmCall struct {
	name string
	args []WasmExpression
	call *ast.CallExpr
}

// ( get_local <var> )
type WasmGetLocal struct {
	astIdent *ast.Ident
	def      WasmVariable
	f        *WasmFunc
	t        *WasmType
}

// ( set_local <var> <expr> )
type WasmSetLocal struct {
	lhs  WasmVariable
	rhs  WasmExpression
	stmt *ast.AssignStmt
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	tok token.Token
	op  BinOp
	x   WasmExpression
	y   WasmExpression
	t   *WasmType
}

func (f *WasmFunc) parseExpr(expr ast.Expr, typeHint *WasmType) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		return nil, fmt.Errorf("unimplemented expression at %s", positionString(expr.Pos(), f.fset))
	case *ast.BasicLit:
		return f.parseBasicLit(expr, typeHint)
	case *ast.BinaryExpr:
		return f.parseBinaryExpr(expr, typeHint)
	case *ast.CallExpr:
		return f.parseCallExpr(expr)
	case *ast.Ident:
		return f.parseIdent(expr)
	case *ast.ParenExpr:
		return f.parseParenExpr(expr, typeHint)
	}
}

func (f *WasmFunc) parseBasicLit(lit *ast.BasicLit, typeHint *WasmType) (WasmExpression, error) {
	if typeHint == nil {
		return nil, fmt.Errorf("not implemented: BasicLit without type hint: %v", lit.Value)
	}
	val := &WasmValue{
		astBasicLiteral: lit,
		t:               typeHint,
	}
	return val, nil
}

func isSupportedBinOp(tok token.Token) bool {
	if int(tok) >= len(binOpMapping) {
		return false
	}
	t := binOpMapping[tok]
	return int(t) > 0
}

func (f *WasmFunc) parseBinaryExpr(expr *ast.BinaryExpr, typeHint *WasmType) (WasmExpression, error) {
	x, err := f.parseExpr(expr.X, typeHint)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand X in a binary expression: %v", err)
	}
	y, err := f.parseExpr(expr.Y, x.getType())
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand Y in a binary expression: %v", err)
	}
	if !isSupportedBinOp(expr.Op) {
		return nil, fmt.Errorf("unsupported binary op: %v", expr.Op)
	}
	xt := x.getType()
	b := &WasmBinOp{
		tok: expr.Op,
		op:  binOpMapping[expr.Op],
		t:   xt,
		x:   x,
		y:   y,
	}
	return b, nil
}

func (f *WasmFunc) parseArgs(args []ast.Expr) []WasmExpression {
	result := make([]WasmExpression, 0, len(args))
	for _, arg := range args {
		e, err := f.parseExpr(arg, nil) // TODO: should this be nil?
		if err != nil {
			panic(err)
		}
		result = append(result, e)
	}
	return result
}

func (f *WasmFunc) parseCallExpr(call *ast.CallExpr) (WasmExpression, error) {
	switch fun := call.Fun.(type) {
	default:
		return nil, fmt.Errorf("unimplemented function: %v at %s", fun, positionString(call.Lparen, f.fset))
	case *ast.Ident:
		args := f.parseArgs(call.Args)
		c := &WasmCall{
			name: astNameToWASM(fun.Name),
			args: args,
			call: call,
		}
		return c, nil
	case *ast.SelectorExpr:
		if isWASMRuntimePackage(fun.X) {
			return f.parseWASMRuntimeCall(fun.Sel, call)
		}
	}
	return nil, fmt.Errorf("call expressions are not implemented at %s", positionString(call.Lparen, f.fset))
}

func (f *WasmFunc) parseIdent(ident *ast.Ident) (WasmExpression, error) {
	v, ok := f.module.variables[ident.Obj]
	if !ok {
		return nil, fmt.Errorf("undefined identifier '%s' at %s", ident.Name, positionString(ident.NamePos, f.fset))
	}
	g := &WasmGetLocal{
		astIdent: ident,
		def:      v,
		f:        f,
	}
	return g, nil
}

func (f *WasmFunc) parseParenExpr(p *ast.ParenExpr, typeHint *WasmType) (WasmExpression, error) {
	return f.parseExpr(p.X, typeHint)
}

func (v *WasmValue) print(writer FormattingWriter) {
	writer.Printf("(")
	v.t.print(writer)
	writer.Printf(".const %s)", v.astBasicLiteral.Value)
}

func (v *WasmValue) getType() *WasmType {
	return v.t
}

func (v *WasmValue) getNode() ast.Node {
	return nil
}

func (b *WasmBinOp) getType() *WasmType {
	return b.t
}

func (b *WasmBinOp) print(writer FormattingWriter) {
	writer.Printf("(")
	b.t.print(writer)
	writer.Printf(".%s ", binOpNames[b.op])
	b.x.print(writer)
	writer.Printf(" ")
	b.y.print(writer)
	writer.Printf(")")
}

func (b *WasmBinOp) getNode() ast.Node {
	return nil
}

func (g *WasmGetLocal) print(writer FormattingWriter) {
	writer.Printf("(get_local %s)", g.def.getName())
}

func (g *WasmGetLocal) getType() *WasmType {
	return g.f.module.variables[g.astIdent.Obj].getType()
}

func (g *WasmGetLocal) getNode() ast.Node {
	return nil
}

func (r *WasmReturn) getType() *WasmType {
	if r.value == nil {
		return nil
	} else {
		return r.value.getType()
	}
}

func (r *WasmReturn) print(writer FormattingWriter) {
	writer.Printf("(return")
	if r.value != nil {
		writer.Printf(" ")
		r.value.print(writer)
	}
	writer.Printf(")")
}

func (r *WasmReturn) getNode() ast.Node {
	if r.stmt == nil {
		return nil
	} else {
		return r.stmt
	}
}

func (s *WasmSetLocal) print(writer FormattingWriter) {
	writer.Printf("(set_local %s ", s.lhs.getName())
	s.rhs.print(writer)
	writer.Printf(")")
}

func (s *WasmSetLocal) getType() *WasmType {
	return s.lhs.getType()
}

func (s *WasmSetLocal) getNode() ast.Node {
	if s.stmt == nil {
		return nil
	} else {
		return s.stmt
	}
}

func (p *WasmCall) getType() *WasmType {
	// TODO
	return nil
}

func (c *WasmCall) print(writer FormattingWriter) {
	writer.Printf("(call %s", c.name)
	for _, arg := range c.args {
		writer.Printf(" ")
		arg.print(writer)
	}
	writer.Printf(")")
}

func (c *WasmCall) getNode() ast.Node {
	if c.call == nil {
		return nil
	} else {
		return c.call
	}
}
