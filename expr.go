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
)

var binOpNames = [...]string{
	binOpAdd: "add",
	binOpSub: "sub",
	binOpMul: "mul",
	binOpDiv: "div_s",
	binOpEq:  "eq",
}

var binOpMapping = [...]BinOp{
	token.ADD: binOpAdd,
	token.SUB: binOpSub,
	token.MUL: binOpMul,
	token.QUO: binOpDiv,
	token.EQL: binOpEq,
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
	astBasicLiteral *ast.BasicLit
	t               *WasmType
}

// Expression list that may introduce new locals, e.g. block or loop.
type WasmScope struct {
	expressions []WasmExpression
	f           *WasmFunc
	indent      int
	n           int
	name        string
}

// ( block <expr>+ )
// ( block <var> <expr>+ ) ;; = (label <var> (block <expr>+))
type WasmBlock struct {
	WasmExprBase
	scope *WasmScope
	stmt  *ast.BlockStmt
}

// ( return <expr>? )
type WasmReturn struct {
	WasmExprBase
	value WasmExpression
	stmt  *ast.ReturnStmt
}

// ( if <expr> <expr> <expr> )
// ( if <expr> <expr> )
type WasmIf struct {
	WasmExprBase
	cond WasmExpression
	body *WasmBlock
	stmt *ast.IfStmt
}

// ( loop <expr>* ) ;; = (loop (block <expr>*))
// ( loop <var> <var>? <expr>* ) ;; = (label <var> (loop (block <var>? <expr>*)))
type WasmLoop struct {
	WasmExprBase
	scope *WasmScope
	cond  WasmExpression
	body  *WasmBlock
	stmt  *ast.ForStmt
}

// ( break <var> <expr>? )
type WasmBreak struct {
	WasmExprBase
	v int
}

// ( call <var> <expr>* )
type WasmCall struct {
	WasmExprBase
	name string
	args []WasmExpression
	call *ast.CallExpr
}

// ( get_local <var> )
type WasmGetLocal struct {
	WasmExprBase
	astIdent *ast.Ident
	def      WasmVariable
	f        *WasmFunc
	t        *WasmType
}

// ( set_local <var> <expr> )
type WasmSetLocal struct {
	WasmExprBase
	lhs  WasmVariable
	rhs  WasmExpression
	stmt ast.Stmt
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	WasmExprBase
	tok token.Token
	op  BinOp
	x   WasmExpression
	y   WasmExpression
	t   *WasmType
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

func (f *WasmFunc) parseExpr(expr ast.Expr, typeHint *WasmType, indent int) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		return nil, fmt.Errorf("unimplemented expression at %s", positionString(expr.Pos(), f.fset))
	case *ast.BasicLit:
		return f.parseBasicLit(expr, typeHint, indent)
	case *ast.BinaryExpr:
		return f.parseBinaryExpr(expr, typeHint, indent)
	case *ast.CallExpr:
		return f.parseCallExpr(expr, indent)
	case *ast.Ident:
		return f.parseIdent(expr, indent)
	case *ast.ParenExpr:
		return f.parseParenExpr(expr, typeHint, indent)
	}
}

func (f *WasmFunc) parseBasicLit(lit *ast.BasicLit, typeHint *WasmType, indent int) (WasmExpression, error) {
	if typeHint == nil {
		return nil, fmt.Errorf("not implemented: BasicLit without type hint: %v", lit.Value)
	}
	val := &WasmValue{
		astBasicLiteral: lit,
		t:               typeHint,
	}
	val.setIndent(indent)
	return val, nil
}

func isSupportedBinOp(tok token.Token) bool {
	if int(tok) >= len(binOpMapping) {
		return false
	}
	t := binOpMapping[tok]
	return int(t) > 0
}

func (f *WasmFunc) parseBinaryExpr(expr *ast.BinaryExpr, typeHint *WasmType, indent int) (WasmExpression, error) {
	x, err := f.parseExpr(expr.X, typeHint, indent+1)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand X in a binary expression: %v", err)
	}
	y, err := f.parseExpr(expr.Y, x.getType(), indent+1)
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
	b.setIndent(indent)
	return b, nil
}

func (f *WasmFunc) parseArgs(args []ast.Expr, indent int) []WasmExpression {
	result := make([]WasmExpression, 0, len(args))
	for _, arg := range args {
		e, err := f.parseExpr(arg, nil, indent) // TODO: Should the hint be nil?
		if err != nil {
			panic(err)
		}
		result = append(result, e)
	}
	return result
}

func (f *WasmFunc) parseConvertExpr(typ string, fun *ast.Ident, v ast.Expr, indent int) (WasmExpression, error) {
	ty, err := f.module.parseAstType(fun)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse type name in type conversion: %v", err)
	}
	return f.parseExpr(v, ty, indent)
}

func (f *WasmFunc) parseCallExpr(call *ast.CallExpr, indent int) (WasmExpression, error) {
	switch fun := call.Fun.(type) {
	default:
		return nil, fmt.Errorf("unimplemented function: %v at %s", fun, positionString(call.Lparen, f.fset))
	case *ast.Ident:
		ty, _, err := f.module.convertAstTypeToWasmType(fun)
		if err == nil && len(call.Args) == 1 {
			return f.parseConvertExpr(ty, fun, call.Args[0], indent)
		}
		args := f.parseArgs(call.Args, indent+1)
		c := &WasmCall{
			name: astNameToWASM(fun.Name, nil),
			args: args,
			call: call,
		}
		c.setIndent(indent)
		return c, nil
	case *ast.SelectorExpr:
		if isWASMRuntimePackage(fun.X) {
			return f.parseWASMRuntimeCall(fun.Sel, call, indent)
		}
	}
	return nil, fmt.Errorf("call expressions are not implemented at %s", positionString(call.Lparen, f.fset))
}

func (f *WasmFunc) parseIdent(ident *ast.Ident, indent int) (WasmExpression, error) {
	v, ok := f.module.variables[ident.Obj]
	if !ok {
		return nil, fmt.Errorf("undefined identifier '%s' at %s", ident.Name, positionString(ident.NamePos, f.fset))
	}
	g := &WasmGetLocal{
		astIdent: ident,
		def:      v,
		f:        f,
	}
	g.setIndent(indent)
	return g, nil
}

func (f *WasmFunc) parseParenExpr(p *ast.ParenExpr, typeHint *WasmType, indent int) (WasmExpression, error) {
	return f.parseExpr(p.X, typeHint, indent)
}

func (v *WasmValue) print(writer FormattingWriter) {
	writer.PrintfIndent(v.getIndent(), "(")
	v.t.print(writer)
	writer.Printf(".const %s)\n", v.astBasicLiteral.Value)
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

func (r *WasmReturn) getType() *WasmType {
	if r.value == nil {
		return nil
	} else {
		return r.value.getType()
	}
}

func (r *WasmReturn) print(writer FormattingWriter) {
	writer.PrintfIndent(r.getIndent(), "(return\n")
	if r.value != nil {
		r.value.print(writer)
	}
	writer.PrintfIndent(r.getIndent(), ") ;; return\n")
}

func (r *WasmReturn) getNode() ast.Node {
	if r.stmt == nil {
		return nil
	} else {
		return r.stmt
	}
}

func (i *WasmIf) getType() *WasmType {
	// TODO
	return nil
}

func (i *WasmIf) print(writer FormattingWriter) {
	writer.PrintfIndent(i.getIndent(), "(if\n")
	i.cond.print(writer)
	writer.Printf("\n")
	i.body.print(writer)
	writer.PrintfIndent(i.getIndent(), ") ;; if\n")
}

func (i *WasmIf) getNode() ast.Node {
	if i.stmt == nil {
		return nil
	} else {
		return i.stmt
	}
}

func (l *WasmLoop) getType() *WasmType {
	// TODO
	return nil
}

func (l *WasmLoop) print(writer FormattingWriter) {
	writer.PrintfIndent(l.getIndent(), "(loop ;; scope %d\n", l.scope.n)
	for _, e := range l.scope.expressions {
		e.print(writer)
	}
	writer.PrintfIndent(l.getIndent(), ") ;; loop\n")
}

func (l *WasmLoop) getNode() ast.Node {
	if l.stmt == nil {
		return nil
	} else {
		return l.stmt
	}
}

func (b *WasmBreak) getType() *WasmType {
	return nil
}

func (b *WasmBreak) print(writer FormattingWriter) {
	writer.PrintfIndent(b.getIndent(), "(break %d)\n", b.v)
}

func (b *WasmBreak) getNode() ast.Node {
	return nil
}

func (s *WasmSetLocal) print(writer FormattingWriter) {
	writer.PrintfIndent(s.getIndent(), "(set_local %s\n", s.lhs.getName())
	s.rhs.print(writer)
	writer.PrintfIndent(s.getIndent(), ") ;; set_local %s\n", s.lhs.getName())
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

func (b *WasmBlock) getType() *WasmType {
	return nil
}

func (b *WasmBlock) print(writer FormattingWriter) {
	writer.PrintfIndent(b.getIndent(), "(block\n")
	for _, expr := range b.scope.expressions {
		expr.print(writer)
	}
	writer.PrintfIndent(b.getIndent(), ") ;; block\n")
}

func (b *WasmBlock) getNode() ast.Node {
	if b.stmt == nil {
		return nil
	} else {
		return b.stmt
	}
}
