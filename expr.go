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
	binOpAnd     BinOp = 11
	binOpOr      BinOp = 12
	binOpXor     BinOp = 13
	binOpShl     BinOp = 14
	binOpShr     BinOp = 15
)

var binOpNames = [...]string{
	binOpAdd: "add",
	binOpSub: "sub",
	binOpMul: "mul",
	binOpDiv: "div",
	binOpEq:  "eq",
	binOpNe:  "ne",
	binOpLt:  "lt",
	binOpLe:  "le",
	binOpGt:  "gt",
	binOpGe:  "ge",
	binOpAnd: "and",
	binOpOr:  "or",
	binOpXor: "xor",
	binOpShl: "shl",
	binOpShr: "shr",
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
	binOpAnd: false,
	binOpOr:  false,
	binOpXor: false,
	binOpShl: false,
	binOpShr: true,
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
	token.AND: binOpAnd,
	token.OR:  binOpOr,
	token.XOR: binOpXor,
	token.SHL: binOpShl,
	token.SHR: binOpShr,
}

type WasmExpression interface {
	print(writer FormattingWriter)
	getType() WasmType
	setType(t WasmType)
	getFullType() WasmType
	setFullType(t WasmType)
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

type LValue struct {
	addr WasmExpression
	t    WasmType
}

type WasmExprBase struct {
	indent   int
	parent   WasmExpression
	astNode  ast.Node
	comment  string
	scope    *WasmScope
	ty       WasmType
	fullType WasmType
}

// value: <int> | <float>
type WasmValue struct {
	WasmExprBase
	value string
}

// ( get_local <var> )
type WasmGetLocal struct {
	WasmExprBase
	astIdent *ast.Ident
	def      WasmVariable
	f        *WasmFunc
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	WasmExprBase
	op BinOp
	x  WasmExpression
	y  WasmExpression
}

// ( <type>.load((8|16)_<sign>)? <offset>? <align>? <expr> )
type WasmLoad struct {
	WasmExprBase
	addr WasmExpression
}

// ( <type>.store <offset>? <align>? <expr> <expr> )
type WasmStore struct {
	WasmExprBase
	addr WasmExpression
	val  WasmExpression
}

func (e *WasmExprBase) getIndent() int {
	return e.indent
}

func (e *WasmExprBase) setIndent(indent int) {
	e.indent = indent
}

func (e *WasmExprBase) setType(t WasmType) {
	e.ty = t
}

func (e *WasmExprBase) getFullType() WasmType {
	return e.fullType
}

func (e *WasmExprBase) setFullType(t WasmType) {
	e.fullType = t
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
		src = e.scope.f.file.getSingleLineGoSource(e.astNode)
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
		return nil, s.f.file.ErrorNode(expr, "unimplemented expression")
	case *ast.BasicLit:
		return s.parseBasicLit(expr, typeHint, indent)
	case *ast.BinaryExpr:
		return s.parseBinaryExpr(expr, typeHint, indent)
	case *ast.CallExpr:
		return s.parseCallExpr(expr, indent)
	case *ast.CompositeLit:
		return s.parseCompositeLit(expr, indent)
	case *ast.Ident:
		return s.parseIdent(expr, indent)
	case *ast.IndexExpr:
		return s.parseIndexExpr(expr, typeHint, indent)
	case *ast.ParenExpr:
		return s.parseParenExpr(expr, typeHint, indent)
	case *ast.SelectorExpr:
		return s.parseSelectorExpr(expr, typeHint, indent)
	case *ast.StarExpr:
		return s.parseStarExpr(expr, typeHint, indent)
	case *ast.UnaryExpr:
		return s.parseUnaryExpr(expr, indent)
	}
}

func (s *WasmScope) createLiteralForType(value int32, typ string, indent int) (WasmExpression, error) {
	t, err := s.f.module.convertAstTypeNameToWasmType(typ)
	if err != nil {
		return nil, fmt.Errorf("couldn't create type %v for a literal: %v", typ, err)
	}
	return s.createLiteral(fmt.Sprintf("%d", value), t, indent)
}

func (s *WasmScope) createLiteralInt32(value int32, indent int) (WasmExpression, error) {
	return s.createLiteralForType(value, "int32", indent)
}

func (s *WasmScope) createNilLiteral(t WasmType, indent int) (WasmExpression, error) {
	switch ty := t.(type) {
	default:
		return s.createLiteral("0", ty, indent)
	case *WasmTypeFunc:
		zero, err := s.createLiteralInt32(-1, indent)
		if err != nil {
			return nil, err
		}
		zero.setComment("nil function pointer")
		zero.setScope(s)
		return zero, nil
	}
}

func (s *WasmScope) createLiteral(value string, ty WasmType, indent int) (WasmExpression, error) {
	if ty == nil {
		return nil, fmt.Errorf("not implemented: literal without type: %v", value)
	}
	val := &WasmValue{
		value: value,
	}
	val.setType(ty)
	val.setIndent(indent)
	return val, nil
}

func (s *WasmScope) parseBasicLit(lit *ast.BasicLit, typeHint WasmType, indent int) (WasmExpression, error) {
	if typeHint == nil {
		switch lit.Kind {
		default:
			return nil, s.f.file.ErrorNode(lit, "not implemented: BasicLit without type hint: %v", lit.Value)
		case token.INT:
			t, err := s.f.file.module.convertAstTypeNameToWasmType("int")
			if err != nil {
				return nil, err
			}
			return s.parseBasicLit(lit, t, indent)
		}
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
		x:  x,
		y:  y,
	}
	b.setType(ty)
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

func (s *WasmScope) parseConvertExpr(ty WasmType, v ast.Expr, indent int) (WasmExpression, error) {
	// TODO: Currently type conversions are nops. We need to check that types have the same size and representation.
	expr, err := s.parseExpr(v, ty, indent)
	if err != nil {
		return nil, err
	}

	switch ty.(type) {
	default:
		expr.setType(ty)
	case *WasmTypePointer:
		// Pointer types are always treated as i32
	}
	expr.setFullType(ty)
	return expr, nil
}

func (s *WasmScope) parseCompositeLit(expr *ast.CompositeLit, indent int) (WasmExpression, error) {
	ty, err := s.f.file.parseAstType(expr.Type)
	if err != nil {
		return nil, fmt.Errorf("CompositeLit, type not found: %v", err)
	}
	switch ty := ty.(type) {
	default:
	case *WasmTypeArray:
		ty.length = uint32(len(expr.Elts))
		size := int32(ty.length) * int32(ty.elementType.getSize())
		align := ty.elementType.getAlign()
		initValue, err := s.generateAlloc(size, int32(align), expr, ty, indent)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate array alloc for CompositeLit: %v", err)
		}
		return initValue, nil
	}
	return nil, fmt.Errorf("unimplemented CompositeLit: %v", expr)
}

func (s *WasmScope) createLoad(addr WasmExpression, t WasmType, indent int) (WasmExpression, error) {
	l := &WasmLoad{
		addr: addr,
	}
	l.setType(t)
	l.setIndent(indent)
	return l, nil
}

func (s *WasmScope) createGetLocal(v WasmVariable, node ast.Node, indent int) *WasmGetLocal {
	g := &WasmGetLocal{
		def: v,
		f:   s.f,
	}
	g.setIndent(indent)
	g.setScope(s)
	g.setNode(node)
	g.setFullType(v.getFullType())
	return g
}

func (s *WasmScope) parseIdent(ident *ast.Ident, indent int) (WasmExpression, error) {
	v, ok := s.f.module.variables[ident.Obj]
	if !ok {
		fn, ok := s.f.module.functionMap2[ident.Obj]
		if !ok {
			return nil, s.f.file.ErrorNode(ident, "undefined identifier '%s'", ident.Name)
		}
		return s.parseFuncIdent(ident, fn, indent)
	}
	switch v := v.(type) {
	default:
		return nil, fmt.Errorf("unimplemented variable kind: %v", v)
	case *WasmGlobalVar:
		addr, err := s.createLiteralInt32(v.addr, indent+1)
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
	g := s.createGetLocal(v, ident, indent)
	g.astIdent = ident
	return g, nil
}

func (s *WasmScope) createIndexExprLValue(index, x WasmExpression, node ast.Node, typeHint WasmType, indent int) (*LValue, error) {
	ty := x.getFullType()
	if ty == nil {
		return nil, s.f.file.ErrorNode(node, "error in IndexExpr: full type of x is nil")
	}
	switch ty := ty.(type) {
	default:
		return nil, fmt.Errorf("unsupported type in IndexExpr: %v", ty)
	case *WasmTypeArray:
		multiplier, err := s.createLiteralInt32(int32(ty.elementType.getSize()), indent+3)
		if err != nil {
			return nil, fmt.Errorf("error in offset for index expression: %v", err)
		}
		multiplier.setComment("array element size")
		offset, err := s.createBinaryExpr(index, multiplier, binOpMul, x.getType(), indent+2)
		if err != nil {
			return nil, fmt.Errorf("error in offset for index expression: %v", err)
		}
		offset.setComment("array element offset")
		addr, err := s.createBinaryExpr(x, offset, binOpAdd, x.getType(), indent+1)
		if err != nil {
			return nil, fmt.Errorf("error in address computation for index expression: %v", err)
		}
		addr.setComment("array element address")
		l := &LValue{
			addr: addr,
			t:    ty.elementType,
		}
		return l, nil
	}
}

func (s *WasmScope) parseIndexExprLValue(expr *ast.IndexExpr, typeHint WasmType, indent int) (*LValue, error) {
	index, err := s.parseExpr(expr.Index, nil, indent+3)
	if err != nil {
		return nil, fmt.Errorf("error in IndexExpr: %v", err)
	}
	index.setComment("array index")
	x, err := s.parseExpr(expr.X, nil, indent+2)
	if err != nil {
		return nil, fmt.Errorf("error in IndexExpr: %v", err)
	}
	return s.createIndexExprLValue(index, x, expr, typeHint, indent)
}

func (s *WasmScope) parseIndexExpr(expr *ast.IndexExpr, typeHint WasmType, indent int) (WasmExpression, error) {
	lvalue, err := s.parseIndexExprLValue(expr, typeHint, indent)
	if err != nil {
		return nil, fmt.Errorf("error in address computation for IndexExpr %v: %v", expr, err)
	}
	l, err := s.createLoad(lvalue.addr, lvalue.t, indent)
	if err != nil {
		return nil, fmt.Errorf("couldn't create a load in a IndexExpr: %v", expr)
	}
	l.setScope(s)
	l.setNode(expr)
	return l, nil
}

func (s *WasmScope) parseParenExpr(p *ast.ParenExpr, typeHint WasmType, indent int) (WasmExpression, error) {
	return s.parseExpr(p.X, typeHint, indent)
}

func (s *WasmScope) createFieldAccessExpr(expr *ast.SelectorExpr, x WasmExpression, field *WasmField, indent int) (*LValue, error) {
	offset, err := s.createLiteralInt32(int32(field.offset), indent+2)
	if err != nil {
		return nil, fmt.Errorf("error in offset for field %s: %v", field.name, err)
	}
	offset.setComment(fmt.Sprintf("field %s, offset: %d", field.name, field.offset))
	addr, err := s.createBinaryExpr(x, offset, binOpAdd, x.getType(), indent+1)
	if err != nil {
		return nil, fmt.Errorf("error in address computation for field %s: %v", field.name, err)
	}
	ptr, err := s.f.file.createPointerType(x.getType())
	if err != nil {
		return nil, fmt.Errorf("couldn't create a type of pointer to: %v", x.getType().getName())
	}
	addr.setFullType(ptr)
	l := &LValue{
		addr: addr,
		t:    x.getType(),
	}
	return l, nil
}

func (s *WasmScope) parseSelectorExprLValue(expr *ast.SelectorExpr, typeHint WasmType, indent int) (*LValue, error) {
	x, err := s.parseExpr(expr.X, nil, indent+2)
	if err != nil {
		return nil, fmt.Errorf("error in SelectorExpr: %v", err)
	}
	ty := x.getFullType()
	if ty == nil {
		return nil, s.f.file.ErrorNode(expr, "error in SelectorExpr: full type of x is nil")
	}
	switch ty := ty.(type) {
	default:
		return nil, fmt.Errorf("unsupported type in SelectorExpr: %v", ty)
	case *WasmTypePointer:
		switch baseTy := ty.base.(type) {
		default:
			return nil, fmt.Errorf("unsupported base type in SelectorExpr: %v", baseTy)
		case *WasmTypeStruct:
			fName := expr.Sel.Name
			for _, f := range baseTy.fields {
				if fName == f.name {
					return s.createFieldAccessExpr(expr, x, f, indent)
				}
			}
			return nil, fmt.Errorf("field %s not found in struct: %v", fName, baseTy)
		}
	}
	return nil, fmt.Errorf("unimplemented SelectorExpr: %v", expr)
}

func (s *WasmScope) parseSelectorExpr(expr *ast.SelectorExpr, typeHint WasmType, indent int) (WasmExpression, error) {
	lvalue, err := s.parseSelectorExprLValue(expr, typeHint, indent)
	if err != nil {
		return nil, fmt.Errorf("error in address computation for SelectorExpr %v: %v", expr, err)
	}
	l, err := s.createLoad(lvalue.addr, lvalue.t, indent)
	if err != nil {
		return nil, fmt.Errorf("couldn't create a load in a SelectorExpr: %v", expr)
	}
	l.setScope(s)
	l.setNode(expr)
	return l, nil
}

func (s *WasmScope) parseExprLValue(expr ast.Expr, typeHint WasmType, indent int) (*LValue, error) {
	switch expr := expr.(type) {
	default:
		return nil, s.f.file.ErrorNode(expr, "unimplemented L-Value expression")
	case *ast.Ident:
		// Assume that identifiers that appear as L-expressions are pointers.
		i, err := s.parseIdent(expr, indent)
		if err != nil {
			return nil, err
		}
		ty := i.getType()
		if ty.getName() == "i32" {
			// TODO: check this is really a pointer type
			lvalue := &LValue{
				addr: i,
				t:    i.getFullType(),
			}
			return lvalue, nil
		}
		return nil, s.f.file.ErrorNode(expr, "unimplemented L-Value Ident expression")
	}
}

func (s *WasmScope) parseStarExpr(expr *ast.StarExpr, typeHint WasmType, indent int) (WasmExpression, error) {
	lvalue, err := s.parseExprLValue(expr.X, typeHint, indent+1)
	if err != nil {
		return nil, err
	}
	l, err := s.createLoad(lvalue.addr, lvalue.t, indent)
	if err != nil {
		return nil, fmt.Errorf("couldn't create a load in a StarExpr: %v", expr)
	}
	l.setScope(s)
	l.setNode(expr)
	return l, nil
}

func (s *WasmScope) generateAlloc(sizeConst, alignConst int32, expr ast.Node, ptrTy WasmType, indent int) (WasmExpression, error) {
	size, err := s.createLiteralInt32(sizeConst, indent+1)
	if err != nil {
		return nil, fmt.Errorf("struct allocation, couldn't create int32 literal for: %v", sizeConst)
	}
	size.setComment("array total size")
	align, err := s.createLiteralInt32(alignConst, indent+1)
	if err != nil {
		return nil, fmt.Errorf("struct allocation, couldn't create int32 literal for: %v", alignConst)
	}
	align.setComment("alignment")
	args := []WasmExpression{size, align}
	allocFnName := mangleFunctionName("gowasm/rt/gc", "Alloc")
	fn, ok := s.f.module.funcSymTab[allocFnName]
	if !ok {
		return nil, fmt.Errorf("link error, couldn't find alloc function: %s", allocFnName)
	}
	callExpr, err := s.createCallExprWithArgs(nil, allocFnName, fn, args, indent)
	if callExpr != nil {
		callExpr.setNode(expr)
		callExpr.setFullType(ptrTy)
	}
	return callExpr, err
}

func (s *WasmScope) parseStructAlloc(expr *ast.CompositeLit, indent int) (WasmExpression, error) {
	t, err := s.f.file.parseAstType(expr.Type)
	if err != nil {
		return nil, fmt.Errorf("struct allocation, type not found: %v", expr.Type)
	}

	ptrTy, err := s.f.file.createPointerType(t)
	if err != nil {
		return nil, fmt.Errorf("struct allocation, couldn't create a pointer type: %v", err)
	}

	return s.generateAlloc(int32(t.getSize()), int32(t.getAlign()), expr, ptrTy, indent)
}

func (s *WasmScope) parseAddressOf(expr ast.Expr, indent int) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		return nil, fmt.Errorf("unsupported address-of operand: %v", expr)
	case *ast.CompositeLit:
		return s.parseStructAlloc(expr, indent)
	case *ast.SelectorExpr:
		lvalue, err := s.parseSelectorExprLValue(expr, nil, indent)
		if err != nil {
			return nil, fmt.Errorf("error in address computation for SelectorExpr %v: %v", expr, err)
		}
		return lvalue.addr, nil
	}
}

func (s *WasmScope) parseBitwiseComplement(astExpr ast.Expr, indent int) (WasmExpression, error) {
	expr, err := s.parseExpr(astExpr, nil, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error in bitwise complement: %v", err)
	}

	// TODO: make it work for int64
	mask, err := s.createLiteral("-1", expr.getType(), indent+1)
	if err != nil {
		return nil, err
	}
	mask.setComment("mask for bitwise complement")
	mask.setScope(s)

	comp, err := s.createBinaryExpr(mask, expr, binOpXor, mask.getType(), indent)
	if err != nil {
		return nil, fmt.Errorf("error in bitwise complement: %v", err)
	}
	return comp, nil
}

func (s *WasmScope) parseUnaryExpr(expr *ast.UnaryExpr, indent int) (WasmExpression, error) {
	switch expr.Op {
	default:
		return nil, fmt.Errorf("unimplemented UnaryExpr, token='%v'", expr.Op)
	case token.AND:
		return s.parseAddressOf(expr.X, indent)
	case token.XOR:
		return s.parseBitwiseComplement(expr.X, indent)
	}
}

func (v *WasmValue) print(writer FormattingWriter) {
	writer.PrintfIndent(v.getIndent(), "(")
	v.ty.print(writer)
	writer.Printf(".const %s)%s\n", v.value, v.getComment())
}

func (v *WasmValue) getType() WasmType {
	return v.ty
}

func (v *WasmValue) getNode() ast.Node {
	return nil
}

func (b *WasmBinOp) getType() WasmType {
	return b.ty
}

func (b *WasmBinOp) print(writer FormattingWriter) {
	writer.PrintfIndent(b.getIndent(), "(")
	b.ty.print(writer)
	writer.Printf(".%s", binOpNames[b.op])
	if binOpWithSign[b.op] && !b.ty.isFloat() {
		if b.ty.isSigned() {
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
	return l.ty
}

func (l *WasmLoad) print(writer FormattingWriter) {
	var ts string
	if l.getType().getSize() == 4 {
		ts = "i32"
	} else {
		panic(fmt.Errorf("uimplemented load type: %v", l.getType()))
	}
	writer.PrintfIndent(l.getIndent(), "(%s.load%s\n", ts, l.getComment())
	l.addr.print(writer)
	writer.PrintfIndent(l.getIndent(), ") ;; load%s\n", l.getComment())
}

func (s *WasmStore) getType() WasmType {
	return s.ty
}

func (s *WasmStore) print(writer FormattingWriter) {
	var ts string
	if s.getType().getSize() == 4 {
		ts = "i32"
	} else {
		panic(fmt.Errorf("uimplemented store type: %v", s.getType()))
	}
	writer.PrintfIndent(s.getIndent(), "(%s.store%s\n", ts, s.getComment())
	s.addr.print(writer)
	s.val.print(writer)
	writer.PrintfIndent(s.getIndent(), ") ;; store%s\n", s.getComment())
}

func (g *WasmGetLocal) print(writer FormattingWriter) {
	writer.PrintfIndent(g.getIndent(), "(get_local %s)%s\n", g.def.getName(), g.getComment())
}

func (g *WasmGetLocal) getType() WasmType {
	if g.ty != nil {
		return g.ty
	} else {
		return g.def.getType()
	}
}
