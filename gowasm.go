package main

// See https://github.com/WebAssembly/spec/tree/master/ml-proto

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

type WasmVariable interface {
	print(writer FormattingWriter)
	getType() *WasmType
	getName() string
}

// module:  ( module <type>* <func>* <global>* <import>* <export>* <table>* <memory>? )
type WasmModule struct {
	f         *ast.File
	fset      *token.FileSet
	indent    int
	name      string
	namePos   token.Pos
	functions []*WasmFunc
	types     map[string]*WasmType
	variables map[*ast.Object]WasmVariable
}

// func:   ( func <name>? <type>? <param>* <result>? <local>* <expr>* )
type WasmFunc struct {
	funcDecl    *ast.FuncDecl
	fset        *token.FileSet
	module      *WasmModule
	indent      int
	name        string
	namePos     token.Pos
	params      []*WasmParam
	result      *WasmResult
	expressions []WasmExpression
}

// param:  ( param <type>* ) | ( param <name> <type> )
type WasmParam struct {
	astIdent *ast.Ident
	astType  ast.Expr
	name     string
	t        *WasmType
}

// result: ( result <type> )
type WasmResult struct {
	astIdent *ast.Ident
	astType  ast.Expr
	name     string
	t        *WasmType
}

// ( return <expr>? )
type WasmReturn struct {
	value WasmExpression
}

// type: i32 | i64 | f32 | f64
type WasmType struct {
	// TODO(cierniak): need an enum here
	name string
	size int
}

// ( get_local <var> )
type WasmGetLocal struct {
	astIdent *ast.Ident
	name     string
	f        *WasmFunc
	t        *WasmType
}

// ( <type>.<binop> <expr> <expr> )
type WasmBinOp struct {
	tok token.Token
	op  BinOp
	x   WasmExpression
	y   WasmExpression
	t   *WasmType
}

func parseAstFile(f *ast.File, fset *token.FileSet) (*WasmModule, error) {
	m := &WasmModule{
		f:         f,
		fset:      fset,
		indent:    0,
		functions: make([]*WasmFunc, 0, 10),
		types:     make(map[string]*WasmType),
		variables: make(map[*ast.Object]WasmVariable),
	}
	if ident := f.Name; ident != nil {
		m.name = ident.Name
		m.namePos = ident.NamePos
	}

	for _, decl := range f.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			fn, err := m.parseAstFuncDecl(funcDecl, fset, m.indent+1)
			if err != nil {
				return nil, err
			}
			m.functions = append(m.functions, fn)
		}
	}
	return m, nil
}

func (m *WasmModule) print(writer FormattingWriter) {
	writer.Printf("(module\n")
	bodyIndent := m.indent + 1
	writer.PrintfIndent(bodyIndent, ";; Go package '%s' %s\n", m.name, writer.SprintPosition(m.namePos, m.fset))
	for _, f := range m.functions {
		writer.Printf("\n")
		f.print(writer)
	}
	writer.Printf(") ;; end Go package '%s'\n", m.name)
}

func (m *WasmModule) parseAstFuncDecl(funcDecl *ast.FuncDecl, fset *token.FileSet, indent int) (*WasmFunc, error) {
	f := &WasmFunc{
		funcDecl:    funcDecl,
		fset:        fset,
		module:      m,
		indent:      indent,
		params:      make([]*WasmParam, 0, 10),
		expressions: make([]WasmExpression, 0, 10),
	}
	if ident := funcDecl.Name; ident != nil {
		f.name = ident.Name
		f.namePos = ident.NamePos
	}
	if funcDecl.Type != nil {
		f.parseType(funcDecl.Type)
	}
	f.parseBody(funcDecl.Body)
	return f, nil
}

func (f *WasmFunc) parseType(t *ast.FuncType) {
	if t.Params.List != nil {
		for _, field := range t.Params.List {
			paramType, err := f.module.parseAstType(field.Type, f.fset)
			if err != nil {
				panic(err)
			}
			for _, name := range field.Names {
				p := &WasmParam{
					astIdent: name,
					astType:  field.Type,
					name:     name.Name,
					t:        paramType,
				}
				f.module.variables[name.Obj] = p
				f.params = append(f.params, p)
			}
		}
	}

	if t.Results != nil {
		if len(t.Results.List) != 1 {
			err := fmt.Errorf("functions returning %d values are not implemented", len(t.Results.List))
			panic(err)
		}
		field := t.Results.List[0]
		paramType, err := f.module.parseAstType(field.Type, f.fset)
		if err != nil {
			panic(err)
		}
		f.result = &WasmResult{
			astType: field.Type,
			t:       paramType,
		}
	}
}

func (f *WasmFunc) parseBody(body *ast.BlockStmt) {
	for _, stmt := range body.List {
		var err error
		var expr WasmExpression
		switch stmt := stmt.(type) {
		default:
			panic(fmt.Errorf("unimplemented statement: %v", stmt))
		case *ast.ReturnStmt:
			expr, err = f.parseReturnStmt(stmt)
		}
		if err != nil {
			panic(err)
		}
		if expr != nil {
			f.expressions = append(f.expressions, expr)
		}
	}
}

func (f *WasmFunc) parseReturnStmt(stmt *ast.ReturnStmt) (WasmExpression, error) {
	r := &WasmReturn{}
	if stmt.Results != nil {
		if len(stmt.Results) != 1 {
			return nil, fmt.Errorf("unimplemented multi-value return statement")
		}
		value, err := f.parseExpr(stmt.Results[0])
		if err != nil {
			return nil, err
		}
		r.value = value
	}
	return r, nil
}

func (f *WasmFunc) parseExpr(expr ast.Expr) (WasmExpression, error) {
	switch expr := expr.(type) {
	default:
		panic(fmt.Errorf("unimplemented expression: %v", expr))
	case *ast.Ident:
		return f.parseIdent(expr)
	case *ast.BinaryExpr:
		return f.parseBinaryExpr(expr)
	}
}

func (f *WasmFunc) parseIdent(ident *ast.Ident) (WasmExpression, error) {
	g := &WasmGetLocal{
		astIdent: ident,
		name:     ident.Name,
		f:        f,
	}
	return g, nil
}

func (f *WasmFunc) parseBinaryExpr(expr *ast.BinaryExpr) (WasmExpression, error) {
	x, err := f.parseExpr(expr.X)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand X in a binary expression", err)
	}
	y, err := f.parseExpr(expr.Y)
	if err != nil {
		return nil, fmt.Errorf("couldn't get operand Y in a binary expression", err)
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

func (f *WasmFunc) print(writer FormattingWriter) {
	writer.PrintfIndent(f.indent, ";; Go function '%s' %s\n", f.name, writer.SprintPosition(f.namePos, f.fset))
	writer.PrintfIndent(f.indent, "(func")
	for _, param := range f.params {
		param.print(writer)
	}
	if f.result != nil {
		f.result.print(writer)
	}
	writer.Printf("\n")
	bodyIndent := f.indent + 1
	for _, expr := range f.expressions {
		writer.PrintfIndent(bodyIndent, "")
		expr.print(writer)
		writer.Printf("\n")
	}
	writer.PrintfIndent(f.indent, ")\n")
}

func (p *WasmParam) getName() string {
	return p.name
}

func (p *WasmParam) getType() *WasmType {
	return p.t
}

func (p *WasmParam) print(writer FormattingWriter) {
	writer.Printf(" (param ")
	if p.name != "" {
		writer.Printf("%s ", p.name)
	}
	p.t.print(writer)
	writer.Printf(")")
}

func (r *WasmResult) print(writer FormattingWriter) {
	writer.Printf(" (result ")
	r.t.print(writer)
	writer.Printf(")")
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
	writer.Printf("(get_local %s)", g.name)
}

func (g *WasmGetLocal) getType() *WasmType {
	return g.f.module.variables[g.astIdent.Obj].getType()
}

func (m *WasmModule) convertAstTypeToWasmType(astType *ast.Ident) (string, int, error) {
	switch astType.Name {
	case "int32":
		return "i32", 32, nil
	case "int64":
		return "i64", 64, nil
	}
	return "", 0, fmt.Errorf("unimplemented type: '%s'", astType.Name)
}

func (m *WasmModule) parseAstType(astType ast.Expr, fset *token.FileSet) (*WasmType, error) {
	if astTypeIdent, ok := astType.(*ast.Ident); ok {
		name := astTypeIdent.Name
		t, ok := m.types[name]
		if !ok {
			typeName, size, err := m.convertAstTypeToWasmType(astTypeIdent)
			if err != nil {
				panic(err)
			}
			t = &WasmType{
				name: typeName,
				size: size,
			}
			m.types[name] = t
		}
		return t, nil
	}
	err := fmt.Errorf("type is not an ident: %v", astType)
	panic(err)
}

func (t *WasmType) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
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
