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
	binOpDiv: "div",
	binOpEq:  "eq",
	binOpNe:  "ne",
	binOpLt:  "lt", // TODO: for floats it should be "lt" and for unsigned, it should be "lt_u".
	binOpLe:  "le",
	binOpGt:  "gt",
	binOpGe:  "ge",
}

var binOpWithSign = [...]bool{
	binOpAdd: false,
	binOpSub: false,
	binOpMul: false,
	binOpDiv: true,
	binOpEq:  false,
	binOpNe:  false,
	binOpLt:  true,
	binOpLe:  true,
	binOpGt:  true,
	binOpGe:  true,
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
	getType() WasmType
	getNode() ast.Node
	setNode(node ast.Node)
	getIndent() int
	setIndent(indent int)
	getParent() WasmExpression
	setParent(parent WasmExpression)
	getComment() string
	setComment(comment string)
	getScope() *WasmScope
	setScope(scope *WasmScope)
}

type WasmExprBase struct {
	indent  int
	parent  WasmExpression
	astNode ast.Node
	comment string
	scope   *WasmScope
}

// value: <int> | <float>
type WasmValue struct {
	WasmExprBase
	value string
	t     WasmType
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
	t        WasmType
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	WasmExprBase
	op BinOp
	x  WasmExpression
	y  WasmExpression
	t  WasmType
}

// ( <type>.load((8|16)_<sign>)? <offset>? <align>? <expr> )
type WasmLoad struct {
	WasmExprBase
	addr WasmExpression
	t    WasmType
}

// ( <type>.store <offset>? <align>? <expr> <expr> )
type WasmStore struct {
	WasmExprBase
	addr WasmExpression
	val  WasmExpression
	t    WasmType
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

func (e *WasmExprBase) getNode() ast.Node {
	return e.astNode
}

func (e *WasmExprBase) setNode(node ast.Node) {
	e.astNode = node
}

func (e *WasmExprBase) getComment() string {
	var result string
	if e.comment != "" || e.astNode != nil {
		result = " ;; "
	}
	var src string
	if e.astNode != nil {
		src = e.scope.f.getSingleLineGoSource(e.astNode)
		result += src
	}
	if e.comment != "" {
		if src != "" {
			result += " // "
		}
		result += e.comment
	}
	return result
}

func (e *WasmExprBase) setComment(comment string) {
	e.comment = comment
}

func (e *WasmExprBase) getScope() *WasmScope {
	return e.scope
}

func (e *WasmExprBase) setScope(scope *WasmScope) {
	e.scope = scope
}

func (s *WasmScope) parseExpr(expr ast.Expr, typeHint WasmType, indent int) (WasmExpression, error) {
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
	case *ast.UnaryExpr:
		return s.parseUnaryExpr(expr, indent)
	}
}

func (s *WasmScope) createLiteral(value string, ty WasmType, indent int) (WasmExpression, error) {
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

func (s *WasmScope) parseBasicLit(lit *ast.BasicLit, typeHint WasmType, indent int) (WasmExpression, error) {
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

func (s *WasmScope) createBinaryExpr(x, y WasmExpression, op BinOp, ty WasmType, indent int) (*WasmBinOp, error) {
	b := &WasmBinOp{
		op: op,
		t:  ty,
		x:  x,
		y:  y,
	}
	b.setIndent(indent)
	b.setScope(s)
	return b, nil
}

func (s *WasmScope) parseBinaryExpr(expr *ast.BinaryExpr, typeHint WasmType, indent int) (WasmExpression, error) {
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
	result, err := s.createBinaryExpr(x, y, binOpMapping[expr.Op], xt, indent)
	if err != nil {
		return nil, fmt.Errorf("couldn't create a binary expression: %v", err)
	}
	result.setNode(expr)
	return result, nil
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
	ty, err := s.f.file.parseAstType(fun)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse type name in type conversion: %v", err)
	}
	return s.parseExpr(v, ty, indent)
}

func (s *WasmScope) createCallExpr(call *ast.CallExpr, name string, fn *WasmFunc, indent int) (WasmExpression, error) {
	args := s.parseArgs(call.Args, indent+1)
	c := &WasmCall{
		name: name,
		args: args,
		call: call,
		def:  fn,
	}
	c.setIndent(indent)
	c.setNode(call)
	c.setScope(s)
	return c, nil
}

func (s *WasmScope) parseCallExpr(call *ast.CallExpr, indent int) (WasmExpression, error) {
	switch fun := call.Fun.(type) {
	default:
		return nil, fmt.Errorf("unimplemented function: %v at %s", fun, positionString(call.Lparen, s.f.fset))
	case *ast.Ident:
		var name string
		fn, ok := s.f.module.functionMap2[fun.Obj]
		if ok {
			name = fn.name
		} else {
			name = astNameToWASM(fun.Name, nil)
		}
		typ, err := s.f.file.parseAstType(fun)
		if err == nil && len(call.Args) == 1 {
			return s.parseConvertExpr(typ.getName(), fun, call.Args[0], indent)
		}

		// TODO: Make it work for forward references.
		decl, ok := fun.Obj.Decl.(*ast.FuncDecl)
		if ok {
			return s.createCallExpr(call, name, s.f.module.functionMap[decl], indent)
		} else {
			return nil, fmt.Errorf("function %s undefined (forward reference?)", name)

		}
	case *ast.SelectorExpr:
		return s.parseCallExprSelector(call, fun, indent)
	}
	return nil, fmt.Errorf("unimplemented call expression at %s", positionString(call.Lparen, s.f.fset))
}

func (s *WasmScope) parseCallExprSelector(call *ast.CallExpr, se *ast.SelectorExpr, indent int) (WasmExpression, error) {
	if isWASMRuntimePackage(se.X) {
		return s.parseWASMRuntimeCall(se.Sel, call, indent)
	}
	switch x := se.X.(type) {
	default:
		return nil, fmt.Errorf("unimplemented X in selector: %v", x)
	case *ast.Ident:
		pkgShort := x.Name
		pkgLong, ok := s.f.file.imports[pkgShort]
		if ok {
			name := mangleFunctionName(pkgLong, se.Sel.Name)
			fn, ok := s.f.module.funcSymTab[name]
			if !ok {
				return nil, fmt.Errorf("link error, couldn't find function: %s", name)
			}
			return s.createCallExpr(call, name, fn, indent)
		}
	}
	return nil, fmt.Errorf("unimplemented selector in a call expression, X: %v, sel: %v", se.X, se.Sel)
}

func (s *WasmScope) createLoad(addr WasmExpression, t WasmType, indent int) (WasmExpression, error) {
	l := &WasmLoad{
		addr: addr,
		t:    t,
	}
	l.setIndent(indent)
	return l, nil
}

func (s *WasmScope) parseIdent(ident *ast.Ident, indent int) (WasmExpression, error) {
	v, ok := s.f.module.variables[ident.Obj]
	if !ok {
		return nil, fmt.Errorf("undefined identifier '%s' at %s", ident.Name, positionString(ident.NamePos, s.f.fset))
	}
	switch v := v.(type) {
	default:
		return nil, fmt.Errorf("unimplemented variable kind: %v", v)
	case *WasmGlobalVar:
		t, err := s.f.module.convertAstTypeNameToWasmType("int32")
		if err != nil {
			return nil, fmt.Errorf("couldn't create address for global %s", v.getName())
		}
		addr, err := s.createLiteral(fmt.Sprintf("%d", v.addr), t, indent+1)
		if err != nil {
			return nil, fmt.Errorf("couldn't create address for global %s", v.getName())
		}
		l, err := s.createLoad(addr, v.getType(), indent)
		if err != nil {
			return nil, fmt.Errorf("couldn't create a load for global %s", v.getName())
		}
		l.setComment(fmt.Sprintf("get_global %s", v.getName()))
		g := &WasmGetGlobal{
			astIdent: ident,
			def:      v,
			f:        s.f,
			load:     l,
		}
		g.setIndent(indent)
		return g, nil
	case *WasmLocal:
	case *WasmParam:
	}
	g := &WasmGetLocal{
		astIdent: ident,
		def:      v,
		f:        s.f,
	}
	g.setIndent(indent)
	g.setScope(s)
	g.setNode(ident)
	return g, nil
}

func (s *WasmScope) parseParenExpr(p *ast.ParenExpr, typeHint WasmType, indent int) (WasmExpression, error) {
	return s.parseExpr(p.X, typeHint, indent)
}

func (s *WasmScope) parseStructAlloc(expr *ast.CompositeLit, indent int) (WasmExpression, error) {
	t, err := s.f.file.parseAstType(expr.Type)
	if err != nil {
		return nil, fmt.Errorf("struct allocation, type not found: %v", expr.Type)
	}
	fmt.Printf("parseStructAlloc, expr: %v, t: %v\n", expr, t)
	return nil, fmt.Errorf("struct allocation is not implemented: %v", expr)
}

func (s *WasmScope) parseAddressOf(expr ast.Expr, indent int) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		return nil, fmt.Errorf("unsupported address-of operand: %v", expr)
	case *ast.CompositeLit:
		return s.parseStructAlloc(expr, indent)
	}
}

func (s *WasmScope) parseUnaryExpr(expr *ast.UnaryExpr, indent int) (WasmExpression, error) {
	switch expr.Op {
	default:
		return nil, fmt.Errorf("unimplemented UnaryExpr, token='%v'", expr.Op)
	case token.AND:
		return s.parseAddressOf(expr.X, indent)
	}
}

func (v *WasmValue) print(writer FormattingWriter) {
	writer.PrintfIndent(v.getIndent(), "(")
	v.t.print(writer)
	writer.Printf(".const %s)\n", v.value)
}

func (v *WasmValue) getType() WasmType {
	return v.t
}

func (v *WasmValue) getNode() ast.Node {
	return nil
}

func (b *WasmBinOp) getType() WasmType {
	return b.t
}

func (b *WasmBinOp) print(writer FormattingWriter) {
	writer.PrintfIndent(b.getIndent(), "(")
	b.t.print(writer)
	writer.Printf(".%s", binOpNames[b.op])
	if binOpWithSign[b.op] {
		if b.t.isSigned() {
			writer.Printf("_s")
		} else {
			writer.Printf("_u")
		}
	}
	writer.Printf("%s\n", b.getComment())
	b.x.print(writer)
	b.y.print(writer)
	writer.PrintfIndent(b.getIndent(), ") ;; bin op %s\n", binOpNames[b.op])
}

func (b *WasmBinOp) getNode() ast.Node {
	return nil
}

func (l *WasmLoad) getType() WasmType {
	return l.t
}

func (l *WasmLoad) print(writer FormattingWriter) {
	var ts string
	if l.getType().getSize() == 32 {
		ts = "i32"
	} else {
		panic(fmt.Errorf("uimplemented load type: %v", l.getType()))
	}
	writer.PrintfIndent(l.getIndent(), "(%s.load%s\n", ts, l.getComment())
	l.addr.print(writer)
	writer.PrintfIndent(l.getIndent(), ") ;; load\n")
}

func (s *WasmStore) getType() WasmType {
	return s.t
}

func (s *WasmStore) print(writer FormattingWriter) {
	var ts string
	if s.getType().getSize() == 32 {
		ts = "i32"
	} else {
		panic(fmt.Errorf("uimplemented store type: %v", s.getType()))
	}
	writer.PrintfIndent(s.getIndent(), "(%s.store%s\n", ts, s.getComment())
	s.addr.print(writer)
	s.val.print(writer)
	writer.PrintfIndent(s.getIndent(), ") ;; load\n")
}

func (g *WasmGetLocal) print(writer FormattingWriter) {
	writer.PrintfIndent(g.getIndent(), "(get_local %s)%s\n", g.def.getName(), g.getComment())
}

func (g *WasmGetLocal) getType() WasmType {
	return g.f.module.variables[g.astIdent.Obj].getType()
}

func (c *WasmCall) getType() WasmType {
	if c.def != nil {
		return c.def.result.t
	}
	return nil
}

func (c *WasmCall) print(writer FormattingWriter) {
	writer.PrintfIndent(c.getIndent(), "(call %s%s\n", c.name, c.getComment())
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
