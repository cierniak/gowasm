package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

type WasmGlobalVar struct {
	name     string
	t        WasmType
	fullType WasmType
	addr     int32
	indent   int
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

func (v *WasmGlobalVar) getFullType() WasmType {
	return v.fullType
}

func (v *WasmGlobalVar) setFullType(t WasmType) {
	v.fullType = t
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

// TODO: refactor parseAstVarDeclGlobal and parseAstVarDeclLocal to share code.
func (file *WasmGoSourceFile) parseAstVarDeclGlobal(decl *ast.GenDecl, fset *token.FileSet) (WasmVariable, error) {
	if len(decl.Specs) != 1 {
		return nil, fmt.Errorf("unsupported variable declaration with %d specs", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return nil, fmt.Errorf("unsupported variable declaration with spec: %v at %s", spec, positionString(spec.Pos(), fset))
	case *ast.ValueSpec:
		return file.parseAstVarSpecGlobal(spec, fset)
	}
}

func (s *WasmScope) parseAstVarDeclLocal(decl *ast.GenDecl) (WasmVariable, error) {
	if len(decl.Specs) != 1 {
		return nil, s.f.file.ErrorNode(decl, "unsupported variable declaration (%d spec)", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return nil, s.f.file.ErrorNode(decl, "unsupported variable declaration")
	case *ast.ValueSpec:
		return s.parseAstVarSpecLocal(spec)
	}
}

func (file *WasmGoSourceFile) initializeGlobalVarBasicLit(v *WasmGlobalVar, lit *ast.BasicLit) error {
	switch lit.Kind {
	default:
		return file.ErrorNode(lit, "unsupported variable initialization of kind %v", lit.Kind)
	case token.INT:
		switch v.getType().getSize() {
		default:
			return file.ErrorNode(lit, "unsupported variable initialization of int of size %d", v.getType().getSize())
		case 4:
			val, err := strconv.ParseInt(lit.Value, 10, 64)
			if err != nil {
				return file.ErrorNode(lit, "couldn't parse int value '%s': %v", lit.Value, err)
			}
			file.module.memory.writeInt32(int(v.addr), int32(val))
			return nil
		}
		return file.ErrorNode(lit, "unsupported INT in variable initialization")
	}
	return file.ErrorNode(lit, "unsupported BasicLit in variable initialization")
}

func (file *WasmGoSourceFile) initializeGlobalVar(v *WasmGlobalVar, spec *ast.ValueSpec) error {
	values := spec.Values
	if values == nil {
		return nil // no initializer
	}
	if len(values) != 1 {
		return file.ErrorNode(spec, fmt.Sprintf("unsupported variable declaration with %d values", len(values)))
	}

	switch val := values[0].(type) {
	default:
		return file.ErrorNode(val, "unsupported variable initialization")
	case *ast.BasicLit:
		return file.initializeGlobalVarBasicLit(v, val)
	}
}

func (file *WasmGoSourceFile) parseAstVarSpecGlobal(spec *ast.ValueSpec, fset *token.FileSet) (WasmVariable, error) {
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
		addr:   int32(file.module.memory.globalVarAddr), // TODO: take alignment into account
		indent: 1,
	}
	file.module.variables[ident.Obj] = v
	file.module.memory.addGlobal(int(v.addr), t.getSize())
	err = file.initializeGlobalVar(v, spec)
	if err != nil {
		return nil, err
	}
	file.module.memory.globalVarAddr += t.getSize()
	if name == "freePointer" {
		// This is a magic name of a global variable used for allocating memory from the heap.
		file.module.freePtrAddr = v.addr
	}
	return v, nil
}

func (s *WasmScope) parseAstVarSpecLocal(spec *ast.ValueSpec) (WasmVariable, error) {
	if len(spec.Names) != 1 {
		return nil, s.f.file.ErrorNode(spec, "unsupported variable declaration (%d names)", len(spec.Names))
	}
	ident := spec.Names[0]
	name := ident.Name
	t, err := s.f.file.parseAstType(spec.Type)
	if err != nil {
		return nil, s.f.file.ErrorNode(spec, "unsupported type for variable %s: %v", name, err)
	}
	return s.createLocalVar(ident, t)
}

func (s *WasmScope) createLocalVar(ident *ast.Ident, ty WasmType) (WasmVariable, error) {
	v := &WasmLocal{
		astIdent: ident,
		name:     astNameToWASM(ident.Name, s),
		t:        ty,
	}
	s.f.module.variables[ident.Obj] = v
	s.f.locals = append(s.f.locals, v)
	return v, nil
}
