package main

import (
	"fmt"
	"go/ast"
)

// type: i32 | i64 | f32 | f64
type WasmType struct {
	name  string
	size  int
	align int
}

func (t *WasmType) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
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

func (m *WasmModule) parseAstType(astType ast.Expr) (*WasmType, error) {
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
