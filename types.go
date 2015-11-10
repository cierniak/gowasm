package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

type WasmTypeI interface {
	getName() string
	getSize() int
	getAlign() int
	print(writer FormattingWriter)
}

// type: i32 | i64 | f32 | f64
type WasmType struct {
	name  string
	size  int
	align int
}

func (t *WasmType) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
}

func (t *WasmType) getName() string {
	return t.name
}

func (t *WasmType) getSize() int {
	return t.size
}

func (t *WasmType) getAlign() int {
	return t.align
}

func (m *WasmModule) convertAstTypeToWasmType(astType *ast.Ident) (*WasmType, error) {
	switch astType.Name {
	case "int32":
		return &WasmType{name: "i32", size: 32, align: 32}, nil
	case "int64":
		return &WasmType{name: "i64", size: 64, align: 64}, nil
	}
	return nil, fmt.Errorf("unimplemented type: '%s'", astType.Name)
}

func (m *WasmModule) parseAstType(astType ast.Expr) (WasmTypeI, error) {
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

func (m *WasmModule) parseAstTypeDecl(decl *ast.GenDecl, fset *token.FileSet) (*WasmType, error) {
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

func (m *WasmModule) parseAstTypeSpec(spec *ast.TypeSpec, fset *token.FileSet) (*WasmType, error) {
	name := spec.Name.Name
	t, ok := m.types[name]
	if !ok {
		t = &WasmType{name: name}
		// Insert incomplete type declaration now to handle recursive types.
		m.types[name] = t
		switch astType := spec.Type.(type) {
		default:
			return nil, fmt.Errorf("unsupported type declaration: %v", astType)
		case *ast.StructType:
			return m.parseAstStructType(t, astType, fset)
		}
	}
	return t, nil
}

func (m *WasmModule) parseAstStructType(t *WasmType, astType *ast.StructType, fset *token.FileSet) (*WasmType, error) {
	fmt.Printf("parseAstStructType, t: %v, astType: %v\n", t, astType)
	return nil, fmt.Errorf("struct types are under construction")
}
