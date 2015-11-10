package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

type WasmType interface {
	getName() string
	getSize() int
	getAlign() int
	print(writer FormattingWriter)
}

type WasmTypeBase struct {
	name  string
	size  int
	align int
}

// type: i32 | i64 | f32 | f64
type WasmTypeScalar struct {
	WasmTypeBase
}

type WasmTypeStruct struct {
	WasmTypeBase
}

func (t *WasmTypeBase) getName() string {
	return t.name
}

func (t *WasmTypeBase) setName(name string) {
	t.name = name
}

func (t *WasmTypeBase) getSize() int {
	return t.size
}

func (t *WasmTypeBase) setSize(size int) {
	t.size = size
}

func (t *WasmTypeBase) getAlign() int {
	return t.align
}

func (t *WasmTypeBase) setAlign(align int) {
	t.align = align
}

func (t *WasmTypeScalar) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
}

func (t *WasmTypeStruct) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
}

func (m *WasmModule) convertAstTypeToWasmType(astType *ast.Ident) (*WasmTypeScalar, error) {
	t := &WasmTypeScalar{}
	switch astType.Name {
	default:
		return nil, fmt.Errorf("unimplemented scalar type: '%s'", astType.Name)
	case "int32":
		t.setName("i32")
		t.setSize(32)
		t.setAlign(32)
	case "int64":
		t.setName("i64")
		t.setSize(64)
		t.setAlign(64)
	}
	return t, nil
}

func (m *WasmModule) parseAstType(astType ast.Expr) (WasmType, error) {
	if astTypeIdent, ok := astType.(*ast.Ident); ok {
		name := astTypeIdent.Name
		t, ok := m.types[name]
		if !ok {
			var err error
			t, err = m.convertAstTypeToWasmType(astTypeIdent)
			if err != nil {
				return nil, err
			}
			m.types[name] = t
		}
		return t, nil
	}
	err := fmt.Errorf("type is not an ident: %v", astType)
	return nil, err
}

func (m *WasmModule) parseAstTypeDecl(decl *ast.GenDecl, fset *token.FileSet) (WasmType, error) {
	if len(decl.Specs) != 1 {
		return nil, fmt.Errorf("unsupported type declaration with %d specs", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return nil, fmt.Errorf("unsupported type declaration with spec: %v at %s", spec, positionString(spec.Pos(), fset))
	case *ast.TypeSpec:
		return m.parseAstTypeSpec(spec, fset)
	}
}

func (m *WasmModule) parseAstTypeSpec(spec *ast.TypeSpec, fset *token.FileSet) (WasmType, error) {
	name := spec.Name.Name
	if t, ok := m.types[name]; ok {
		return t, nil
	}
	switch astType := spec.Type.(type) {
	default:
		return nil, fmt.Errorf("unsupported type declaration: %v", astType)
	case *ast.StructType:
		st := &WasmTypeStruct{}
		st.setName(name)
		// Insert incomplete the type declaration now to handle recursive types.
		m.types[name] = st
		return m.parseAstStructType(st, astType, fset)
	}
}

func (m *WasmModule) parseAstStructType(t *WasmTypeStruct, astType *ast.StructType, fset *token.FileSet) (WasmType, error) {
	fmt.Printf("parseAstStructType, t: %v, astType: %v\n", t, astType)
	return nil, fmt.Errorf("struct types are under construction")
}
