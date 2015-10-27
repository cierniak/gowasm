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
)

var binOpNames = [...]string{
	binOpAdd: "add",
}

var binOpMapping = [...]BinOp{
	token.ADD: binOpAdd,
}

type WasmExpression interface {
	print(writer FormattingWriter)
	getType() *WasmType
}

// ( return <expr>? )
type WasmReturn struct {
	value WasmExpression
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
	lhs WasmVariable
	rhs WasmExpression
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	tok token.Token
	op  BinOp
	x   WasmExpression
	y   WasmExpression
	t   *WasmType
}

func (f *WasmFunc) parseExpr(expr ast.Expr) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		return nil, fmt.Errorf("unimplemented expression at %s", positionString(expr.Pos(), f.fset))
	case *ast.BinaryExpr:
		return f.parseBinaryExpr(expr)
	case *ast.CallExpr:
		return f.parseCallExpr(expr)
	case *ast.Ident:
		return f.parseIdent(expr)
	}
}

func (f *WasmFunc) parseBinaryExpr(expr *ast.BinaryExpr) (WasmExpression, error) {
	x, err := f.parseExpr(expr.X)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand X in a binary expression: %v", err)
	}
	y, err := f.parseExpr(expr.Y)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand Y in a binary expression: %v", err)
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
		e, err := f.parseExpr(arg)
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

func (g *WasmGetLocal) print(writer FormattingWriter) {
	writer.Printf("(get_local %s)", g.def.getName())
}

func (g *WasmGetLocal) getType() *WasmType {
	return g.f.module.variables[g.astIdent.Obj].getType()
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

func (s *WasmSetLocal) print(writer FormattingWriter) {
	writer.Printf("(set_local %s ", s.lhs.getName())
	s.rhs.print(writer)
	writer.Printf(")")
}

func (s *WasmSetLocal) getType() *WasmType {
	return s.lhs.getType()
}
