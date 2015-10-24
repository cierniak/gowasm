package main

// See https://github.com/WebAssembly/spec/tree/master/ml-proto

import (
	"fmt"
	"go/ast"
	"go/token"
)

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
	imports   map[string]*WasmImport
}

func parseAstFile(f *ast.File, fset *token.FileSet) (*WasmModule, error) {
	m := &WasmModule{
		f:         f,
		fset:      fset,
		indent:    0,
		functions: make([]*WasmFunc, 0, 10),
		types:     make(map[string]*WasmType),
		variables: make(map[*ast.Object]WasmVariable),
		imports:   make(map[string]*WasmImport),
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

func (m *WasmModule) printImports(writer FormattingWriter, indent int) {
	for _, i := range m.imports {
		i.print(writer)
	}
}

func (m *WasmModule) print(writer FormattingWriter) {
	writer.Printf("(module\n")
	bodyIndent := m.indent + 1
	writer.PrintfIndent(bodyIndent, ";; Go package '%s' %s\n", m.name, positionString(m.namePos, m.fset))
	m.printImports(writer, bodyIndent)
	for _, f := range m.functions {
		writer.Printf("\n")
		f.print(writer)
	}
	writer.Printf(") ;; end Go package '%s'\n", m.name)
}

func astNameToWASM(astName string) string {
	return "$" + astName
}

func positionString(pos token.Pos, fset *token.FileSet) string {
	position := fset.File(pos).PositionFor(pos, false)
	return fmt.Sprintf("[%s:%d:%d]", position.Filename, position.Line, position.Offset)
}
