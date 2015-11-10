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
	binOpDiv     BinOp = 4
	binOpEq      BinOp = 5
	binOpNe      BinOp = 6
	binOpLt      BinOp = 7
	binOpLe      BinOp = 8
	binOpGt      BinOp = 9
	binOpGe      BinOp = 10
)

var binOpNames = [...]string{
	binOpAdd: "add",
	binOpSub: "sub",
	binOpMul: "mul",
	binOpDiv: "div_s",
	binOpEq:  "eq",
	binOpNe:  "ne",
	binOpLt:  "lt_s", // TODO: for floats it should be "lt" and for unsigned, it should be "lt_u".
	binOpLe:  "le_s",
	binOpGt:  "gt_s",
	binOpGe:  "ge_s",
}

var binOpMapping = [...]BinOp{
	token.ADD: binOpAdd,
	token.SUB: binOpSub,
	token.MUL: binOpMul,
	token.QUO: binOpDiv,
	token.EQL: binOpEq,
	token.NEQ: binOpNe,
	token.LSS: binOpLt,
	token.LEQ: binOpLe,
	token.GTR: binOpGt,
	token.GEQ: binOpGe,
	token.INC: binOpAdd,
	token.DEC: binOpSub,
}

type WasmExpression interface {
	print(writer FormattingWriter)
	getType() *WasmType
	getNode() ast.Node
	getIndent() int
	setIndent(indent int)
	getParent() WasmExpression
	setParent(parent WasmExpression)
}

type WasmExprBase struct {
	indent int
	parent WasmExpression
}

// value: <int> | <float>
type WasmValue struct {
	WasmExprBase
	value string
	t     *WasmType
}

// ( call <var> <expr>* )
type WasmCall struct {
	WasmExprBase
	name string
	args []WasmExpression
	call *ast.CallExpr
	def  *WasmFunc
}

// ( get_local <var> )
type WasmGetLocal struct {
	WasmExprBase
	astIdent *ast.Ident
	def      WasmVariable
	f        *WasmFunc
	t        *WasmType
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	WasmExprBase
	op BinOp
	x  WasmExpression
	y  WasmExpression
	t  *WasmType
}

func (e *WasmExprBase) getIndent() int {
	return e.indent
}

func (e *WasmExprBase) setIndent(indent int) {
	e.indent = indent
}

func (e *WasmExprBase) getParent() WasmExpression {
	return e.parent
}

func (e *WasmExprBase) setParent(parent WasmExpression) {
	e.parent = parent
}

func (s *WasmScope) parseExpr(expr ast.Expr, typeHint *WasmType, indent int) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		return nil, fmt.Errorf("unimplemented expression at %s", positionString(expr.Pos(), s.f.fset))
	case *ast.BasicLit:
		return s.parseBasicLit(expr, typeHint, indent)
	case *ast.BinaryExpr:
		return s.parseBinaryExpr(expr, typeHint, indent)
	case *ast.CallExpr:
		return s.parseCallExpr(expr, indent)
	case *ast.Ident:
		return s.parseIdent(expr, indent)
	case *ast.ParenExpr:
		return s.parseParenExpr(expr, typeHint, indent)
	}
}

func (s *WasmScope) createLiteral(value string, ty *WasmType, indent int) (WasmExpression, error) {
	if ty == nil {
		return nil, fmt.Errorf("not implemented: literal without type: %v", value)
	}
	val := &WasmValue{
		value: value,
		t:     ty,
	}
	val.setIndent(indent)
	return val, nil
}

func (s *WasmScope) parseBasicLit(lit *ast.BasicLit, typeHint *WasmType, indent int) (WasmExpression, error) {
	if typeHint == nil {
		return nil, fmt.Errorf("not implemented: BasicLit without type hint: %v", lit.Value)
	}
	return s.createLiteral(lit.Value, typeHint, indent)
}

func isSupportedBinOp(tok token.Token) bool {
	if int(tok) >= len(binOpMapping) {
		return false
	}
	t := binOpMapping[tok]
	return int(t) > 0
}

func (s *WasmScope) createBinaryExpr(x, y WasmExpression, op BinOp, ty *WasmType, indent int) (*WasmBinOp, error) {
	b := &WasmBinOp{
		op: op,
		t:  ty,
		x:  x,
		y:  y,
	}
	b.setIndent(indent)
	return b, nil
}

func (s *WasmScope) parseBinaryExpr(expr *ast.BinaryExpr, typeHint *WasmType, indent int) (WasmExpression, error) {
	x, err := s.parseExpr(expr.X, typeHint, indent+1)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand X in a binary expression: %v", err)
	}
	y, err := s.parseExpr(expr.Y, x.getType(), indent+1)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand Y in a binary expression: %v", err)
	}
	if !isSupportedBinOp(expr.Op) {
		return nil, fmt.Errorf("unsupported binary op: %v", expr.Op)
	}
	xt := x.getType()
	return s.createBinaryExpr(x, y, binOpMapping[expr.Op], xt, indent)
}

func (s *WasmScope) parseArgs(args []ast.Expr, indent int) []WasmExpression {
	result := make([]WasmExpression, 0, len(args))
	for _, arg := range args {
		e, err := s.parseExpr(arg, nil, indent) // TODO: Should the hint be nil?
		if err != nil {
			panic(err)
		}
		result = append(result, e)
	}
	return result
}

func (s *WasmScope) parseConvertExpr(typ string, fun *ast.Ident, v ast.Expr, indent int) (WasmExpression, error) {
	ty, err := s.f.module.parseAstType(fun)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse type name in type conversion: %v", err)
	}
	return s.parseExpr(v, ty, indent)
}

func (s *WasmScope) parseCallExpr(call *ast.CallExpr, indent int) (WasmExpression, error) {
	switch fun := call.Fun.(type) {
	default:
		return nil, fmt.Errorf("unimplemented function: %v at %s", fun, positionString(call.Lparen, s.f.fset))
	case *ast.Ident:
		typ, err := s.f.module.parseAstType(fun)
		if err == nil && len(call.Args) == 1 {
			return s.parseConvertExpr(typ.name, fun, call.Args[0], indent)
		}
		args := s.parseArgs(call.Args, indent+1)
		c := &WasmCall{
			name: astNameToWASM(fun.Name, nil),
			args: args,
			call: call,
		}
		// TODO: Make it work for forward references.
		decl, ok := fun.Obj.Decl.(*ast.FuncDecl)
		if ok {
			def, ok := s.f.module.functionMap[decl]
			if ok {
				c.def = def
			}

		}
		c.setIndent(indent)
		return c, nil
	case *ast.SelectorExpr:
		if isWASMRuntimePackage(fun.X) {
			return s.parseWASMRuntimeCall(fun.Sel, call, indent)
		}
	}
	return nil, fmt.Errorf("unimplemented call expression at %s", positionString(call.Lparen, s.f.fset))
}

func (s *WasmScope) parseIdent(ident *ast.Ident, indent int) (WasmExpression, error) {
	v, ok := s.f.module.variables[ident.Obj]
	if !ok {
		return nil, fmt.Errorf("undefined identifier '%s' at %s", ident.Name, positionString(ident.NamePos, s.f.fset))
	}
	g := &WasmGetLocal{
		astIdent: ident,
		def:      v,
		f:        s.f,
	}
	g.setIndent(indent)
	return g, nil
}

func (s *WasmScope) parseParenExpr(p *ast.ParenExpr, typeHint *WasmType, indent int) (WasmExpression, error) {
	return s.parseExpr(p.X, typeHint, indent)
}

func (v *WasmValue) print(writer FormattingWriter) {
	writer.PrintfIndent(v.getIndent(), "(")
	v.t.print(writer)
	writer.Printf(".const %s)\n", v.value)
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
	writer.PrintfIndent(b.getIndent(), "(")
	b.t.print(writer)
	writer.Printf(".%s\n", binOpNames[b.op])
	b.x.print(writer)
	b.y.print(writer)
	writer.PrintfIndent(b.getIndent(), ") ;; bin op %s\n", binOpNames[b.op])
}

func (b *WasmBinOp) getNode() ast.Node {
	return nil
}

func (g *WasmGetLocal) print(writer FormattingWriter) {
	writer.PrintfIndent(g.getIndent(), "(get_local %s)\n", g.def.getName())
}

func (g *WasmGetLocal) getType() *WasmType {
	return g.f.module.variables[g.astIdent.Obj].getType()
}

func (g *WasmGetLocal) getNode() ast.Node {
	return nil
}

func (c *WasmCall) getType() *WasmType {
	if c.def != nil {
		return c.def.result.t
	}
	return nil
}

func (c *WasmCall) print(writer FormattingWriter) {
	writer.PrintfIndent(c.getIndent(), "(call %s\n", c.name)
	for _, arg := range c.args {
		arg.print(writer)
	}
	writer.PrintfIndent(c.getIndent(), ") ;; call %s\n", c.name)
}

func (c *WasmCall) getNode() ast.Node {
	if c.call == nil {
		return nil
	} else {
		return c.call
	}
}
