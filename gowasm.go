package main

// See https://github.com/WebAssembly/spec/tree/master/ml-proto

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type WasmVariable interface {
	print(writer FormattingWriter)
	getType() WasmType
	getName() string
}

// module:  ( module <type>* <func>* <global>* <import>* <export>* <table>* <memory>? )
type WasmModule struct {
	f            *ast.File
	fset         *token.FileSet
	indent       int
	name         string
	namePos      token.Pos
	functions    []*WasmFunc
	functionMap  map[*ast.FuncDecl]*WasmFunc
	types        map[string]WasmType
	variables    map[*ast.Object]WasmVariable
	imports      map[string]*WasmImport
	assertReturn []string
	invoke       []string
}

func parseAstFile(f *ast.File, fset *token.FileSet) (*WasmModule, error) {
	m := &WasmModule{
		f:            f,
		fset:         fset,
		indent:       0,
		functions:    make([]*WasmFunc, 0, 10),
		functionMap:  make(map[*ast.FuncDecl]*WasmFunc),
		types:        make(map[string]WasmType),
		variables:    make(map[*ast.Object]WasmVariable),
		imports:      make(map[string]*WasmImport),
		assertReturn: make([]string, 0, 10),
		invoke:       make([]string, 0, 10),
	}
	if ident := f.Name; ident != nil {
		m.name = ident.Name
		m.namePos = ident.NamePos
	}

	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		default:
			return nil, fmt.Errorf("unimplemented declaration type: %v at %s", decl, positionString(decl.Pos(), fset))
		case *ast.GenDecl:
			switch decl.Tok {
			default:
				fmt.Printf("Ignoring GenDecl, token: %v\n", decl.Tok)
			case token.TYPE:
				_, err := m.parseAstTypeDecl(decl, fset)
				if err != nil {
					return nil, err
				}
			}
		case *ast.FuncDecl:
			fn, err := m.parseAstFuncDecl(decl, fset, m.indent+1)
			if err != nil {
				return nil, err
			}
			m.functions = append(m.functions, fn)
			m.functionMap[decl] = fn
		}
	}
	return m, nil
}

func (m *WasmModule) parseComment(c string) {
	pragmaPrefix := "//wasm:"
	if strings.HasPrefix(c, pragmaPrefix) {
		m.parsePragma(strings.TrimPrefix(c, pragmaPrefix))
	}
}

func (m *WasmModule) parsePragma(p string) {
	assertReturnPrefix := "assert_return "
	invokePrefix := "invoke "
	if strings.HasPrefix(p, assertReturnPrefix) {
		m.assertReturn = append(m.assertReturn, strings.TrimPrefix(p, assertReturnPrefix))
	} else if strings.HasPrefix(p, invokePrefix) {
		m.invoke = append(m.invoke, strings.TrimPrefix(p, invokePrefix))
	}
}

func (m *WasmModule) printImports(writer FormattingWriter) {
	for _, i := range m.imports {
		i.print(writer)
	}
}

func isSymbolPublic(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func (m *WasmModule) printExports(writer FormattingWriter, indent int) {
	for _, f := range m.functions {
		if isSymbolPublic(f.origName) {
			writer.PrintfIndent(indent, "(export \"%s\" %s)\n", f.origName, f.name)
		}
	}
}

func (m *WasmModule) print(writer FormattingWriter) {
	writer.Printf("(module\n")
	bodyIndent := m.indent + 1
	writer.PrintfIndent(bodyIndent, ";; Go package '%s' %s\n", m.name, positionString(m.namePos, m.fset))
	m.printImports(writer)
	for _, f := range m.functions {
		writer.Printf("\n")
		f.print(writer)
	}
	writer.Printf("\n")
	m.printExports(writer, bodyIndent)
	writer.Printf(") ;; end Go package '%s'\n", m.name)
	writer.Printf("\n")
	for _, a := range m.assertReturn {
		writer.PrintfIndent(m.indent, "(assert_return %s)\n", a)
	}
	for _, a := range m.invoke {
		writer.PrintfIndent(m.indent, "%s\n", a)
	}
}

func astNameToWASM(astName string, s *WasmScope) string {
	if s == nil {
		return "$" + astName
	} else {
		return fmt.Sprintf("$%s_%s", s.name, astName)
	}
}

func positionString(pos token.Pos, fset *token.FileSet) string {
	position := fset.File(pos).PositionFor(pos, false)
	return fmt.Sprintf("[%s:%d:%d]", position.Filename, position.Line, position.Offset)
}
