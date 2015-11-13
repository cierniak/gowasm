package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

type WasmGlobalVar struct {
	name   string
	t      WasmType
	addr   int32
	indent int
}

type WasmGetGlobal struct {
	WasmExprBase
	astIdent *ast.Ident
	def      WasmVariable
	f        *WasmFunc
	t        WasmType
	load     WasmExpression
}

type WasmSetGlobal struct {
	WasmExprBase
	lhs   WasmVariable
	rhs   WasmExpression
	stmt  ast.Stmt
	store WasmExpression
}

func (v *WasmGlobalVar) print(writer FormattingWriter) {
	writer.PrintfIndent(v.indent, ";; @%x (size %d): var %s %s\n", v.addr, v.t.getSize(), v.getName(), v.t.getName())
}

func (v *WasmGlobalVar) getType() WasmType {
	return v.t
}

func (v *WasmGlobalVar) getName() string {
	return v.name
}

func (g *WasmGetGlobal) print(writer FormattingWriter) {
	g.load.print(writer)
}

func (g *WasmGetGlobal) getType() WasmType {
	return g.f.module.variables[g.astIdent.Obj].getType()
}

func (g *WasmGetGlobal) getNode() ast.Node {
	return nil
}

func (s *WasmSetGlobal) print(writer FormattingWriter) {
	s.store.print(writer)
}

func (s *WasmSetGlobal) getType() WasmType {
	return s.lhs.getType()
}

func (s *WasmSetGlobal) getNode() ast.Node {
	if s.stmt == nil {
		return nil
	} else {
		return s.stmt
	}
}

func (file *WasmGoSourceFile) parseAstVarDecl(decl *ast.GenDecl, fset *token.FileSet) (WasmVariable, error) {
	if len(decl.Specs) != 1 {
		return nil, fmt.Errorf("unsupported variable declaration with %d specs", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return nil, fmt.Errorf("unsupported variable declaration with spec: %v at %s", spec, positionString(spec.Pos(), fset))
	case *ast.ValueSpec:
		return file.parseAstVarSpec(spec, fset)
	}
}

func (file *WasmGoSourceFile) parseAstVarSpec(spec *ast.ValueSpec, fset *token.FileSet) (WasmVariable, error) {
	if len(spec.Names) != 1 {
		return nil, fmt.Errorf("unsupported variable declaration with %d names", len(spec.Names))
	}
	ident := spec.Names[0]
	name := ident.Name
	t, err := file.parseAstType(spec.Type)
	if err != nil {
		return nil, fmt.Errorf("unsupported type for variable %s", name)
	}
	v := &WasmGlobalVar{
		name:   name,
		t:      t,
		addr:   int32(file.module.globalVarAddr), // TODO: take alignment into account
		indent: 1,
	}
	file.module.variables[ident.Obj] = v
	file.module.globalVarAddr += t.getSize()
	if name == "freePointer" {
		// This is a magic name of a global variable used for allocating memory from the heap.
		file.module.freePtrAddr = v.addr
	}
	return v, nil
}
