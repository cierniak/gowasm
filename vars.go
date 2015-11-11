package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

type WasmGlobalVar struct {
	name string
	t    WasmType
	addr int32
}

func (v *WasmGlobalVar) print(writer FormattingWriter) {
	writer.Printf(";; global variable %s\n", v.getName())
}

func (v *WasmGlobalVar) getType() WasmType {
	return v.t
}

func (v *WasmGlobalVar) getName() string {
	return v.name
}

func (m *WasmModule) parseAstVarDecl(decl *ast.GenDecl, fset *token.FileSet) (WasmVariable, error) {
	if len(decl.Specs) != 1 {
		return nil, fmt.Errorf("unsupported variable declaration with %d specs", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return nil, fmt.Errorf("unsupported variable declaration with spec: %v at %s", spec, positionString(spec.Pos(), fset))
	case *ast.ValueSpec:
		return m.parseAstVarSpec(spec, fset)
	}
}

func (m *WasmModule) parseAstVarSpec(spec *ast.ValueSpec, fset *token.FileSet) (WasmVariable, error) {
	if len(spec.Names) != 1 {
		return nil, fmt.Errorf("unsupported variable declaration with %d names", len(spec.Names))
	}
	ident := spec.Names[0]
	name := ident.Name
	t, err := m.parseAstType(spec.Type)
	if err != nil {
		return nil, fmt.Errorf("unsupported type for variable %s", name)
	}
	v := &WasmGlobalVar{
		name: name,
		t:    t,
	}
	m.variables[ident.Obj] = v
	fmt.Printf("parseAstVarSpec, v: %v\n", v)
	return v, nil
}
