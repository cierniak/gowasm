package main

import (
	"fmt"
	"go/ast"
)

// type: i32 | i64 | f32 | f64
type WasmType struct {
	// TODO(cierniak): need an enum here
	name string
	size int
}

func (t *WasmType) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
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

func (m *WasmModule) parseAstType(astType ast.Expr) (*WasmType, error) {
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
