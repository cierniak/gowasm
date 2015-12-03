package main

import (
	"fmt"
	"go/ast"
)

type WasmType interface {
	getName() string
	setName(name string)
	getSize() int
	getAlign() int
	isSigned() bool
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
	signed bool
}

type WasmTypePointer struct {
	WasmTypeBase
	base WasmType
}

type WasmField struct {
	name   string
	offset int
	t      WasmType
}

type WasmTypeStruct struct {
	WasmTypeBase
	fields []*WasmField
}

type WasmTypeFunc struct {
	WasmTypeBase
	wasmName string
	result   WasmType
	indent   int
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

func (t *WasmTypeScalar) isSigned() bool {
	return t.signed
}

func (t *WasmTypeScalar) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
}

func (t *WasmTypeFunc) isSigned() bool {
	return false
}

func (t *WasmTypeFunc) printType(writer FormattingWriter) {
	writer.PrintfIndent(t.indent, "(type %s (func (param)", t.wasmName)
	if t.result != nil {
		writer.Printf(" (result ")
		t.result.print(writer)
		writer.Printf(")")
	}
	writer.Printf("))")
	if t.name != "" {
		writer.Printf(" ;; %s", t.name)
	}
	writer.Printf("\n")
}

func (t *WasmTypeFunc) print(writer FormattingWriter) {
	writer.Printf("i32")
}

func (t *WasmTypePointer) isSigned() bool {
	return false
}

func (t *WasmTypePointer) print(writer FormattingWriter) {
	writer.Printf("i32")
}

func (t *WasmTypeStruct) isSigned() bool {
	return false
}

func (t *WasmTypeStruct) print(writer FormattingWriter) {
	writer.Printf("%s", t.name)
}

func (m *WasmModule) convertAstTypeNameToWasmType(name string) (*WasmTypeScalar, error) {
	t := &WasmTypeScalar{}
	switch name {
	default:
		return nil, fmt.Errorf("unimplemented scalar type: '%s'", name)
	case "int32":
		t.setName("i32")
		t.setSize(4)
		t.setAlign(4)
		t.signed = true
	case "int64":
		t.setName("i64")
		t.setSize(8)
		t.setAlign(8)
		t.signed = true
	case "int":
		fallthrough
	case "uint32":
		fallthrough
	case "unsafe.Pointer":
		fallthrough
	case "uintptr":
		t.setName("i32")
		t.setSize(4)
		t.setAlign(4)
		t.signed = false
	}
	return t, nil
}

func (file *WasmGoSourceFile) convertAstTypeToWasmType(astType *ast.Ident) (*WasmTypeScalar, error) {
	return file.module.convertAstTypeNameToWasmType(astType.Name)
}

func (file *WasmGoSourceFile) parseAstType(astType ast.Expr) (WasmType, error) {
	switch astType := astType.(type) {
	default:
		return nil, fmt.Errorf("unsupported type: %v", astType)
	case *ast.Ident:
		name := astType.Name
		t, ok := file.module.types[name]
		if !ok {
			var err error
			t, err = file.convertAstTypeToWasmType(astType)
			if err != nil {
				return nil, err
			}
			file.module.types[name] = t
		}
		return t, nil
	case *ast.StarExpr:
		base, err := file.parseAstType(astType.X)
		if err != nil {
			return nil, fmt.Errorf("error in a pointer type: %v", err)
		}
		return file.createPointerType(base)
	}
}

func (file *WasmGoSourceFile) createPointerType(t WasmType) (WasmType, error) {
	ptrTy := &WasmTypePointer{
		base: t,
	}
	ptrTy.setName(fmt.Sprintf("*%s", t.getName()))
	ptrTy.setAlign(4)
	ptrTy.setSize(4)
	return ptrTy, nil
}

func (file *WasmGoSourceFile) parseAstTypeDecl(decl *ast.GenDecl) (WasmType, error) {
	if len(decl.Specs) != 1 {
		return nil, fmt.Errorf("unsupported type declaration with %d specs", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return nil, fmt.Errorf("unsupported type declaration with spec: %v at %s", spec, positionString(spec.Pos(), file.fset))
	case *ast.TypeSpec:
		return file.parseAstTypeSpec(spec)
	}
}

func (file *WasmGoSourceFile) parseAstTypeSpec(spec *ast.TypeSpec) (WasmType, error) {
	name := spec.Name.Name
	if t, ok := file.module.types[name]; ok {
		return t, nil
	}
	switch astType := spec.Type.(type) {
	default:
		return nil, file.ErrorNode(spec, "unsupported type declaration: %v", astType)
	case *ast.FuncType:
		ty, err := file.parseAstFuncType(astType)
		if err != nil {
			return nil, err
		}
		ty.setName(name)
		file.module.types[name] = ty
		return ty, nil
	case *ast.StructType:
		st := &WasmTypeStruct{}
		st.setName(name)
		st.setAlign(8)
		// Insert incomplete the type declaration now to handle recursive types.
		file.module.types[name] = st
		return file.parseAstStructType(st, astType)
	}
}

func (file *WasmGoSourceFile) parseAstStructType(t *WasmTypeStruct, astType *ast.StructType) (WasmType, error) {
	if astType.Fields == nil || astType.Fields.List == nil {
		return nil, fmt.Errorf("struct types with no fields are not supported (struct %s)", t.getName())
	}
	astFields := astType.Fields.List
	t.fields = make([]*WasmField, len(astFields), len(astFields))
	var offset int
	for i, astField := range astFields {
		if len(astField.Names) != 1 {
			return nil, fmt.Errorf("struct types with multiple fields per type are not supported (struct %s)", t.getName())
		}
		field := &WasmField{
			name:   astField.Names[0].Name,
			offset: offset,
		}
		t.fields[i] = field
		ty, err := file.parseAstType(astField.Type)
		if err != nil {
			return nil, fmt.Errorf("error parsing type of field %s: %v", field.name, err)
		}
		field.t = ty
		offset += ty.getSize() // TODO: Take alignment into account
	}
	t.setSize(offset)
	return t, nil
}
