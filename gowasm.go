package main

// See https://github.com/WebAssembly/spec/tree/master/ml-proto

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type GoWasmError struct {
	node ast.Node
	msg  string
}

type WasmModuleLinker interface {
	addAstFile(f *ast.File, fset *token.FileSet) error
	finalize() error
	print(writer FormattingWriter)
}

type WasmVariable interface {
	print(writer FormattingWriter)
	getType() WasmType
	getFullType() WasmType
	setFullType(t WasmType)
	getName() string
}

// module:  ( module <type>* <func>* <global>* <import>* <export>* <table>* <memory>? )
type WasmModule struct {
	indent       int
	name         string
	namePos      token.Pos
	files        []*WasmGoSourceFile
	functions    []*WasmFunc
	functionMap  map[*ast.FuncDecl]*WasmFunc
	functionMap2 map[*ast.Object]*WasmFunc
	funcSymTab   map[string]*WasmFunc
	funcPtrTable *WasmFunctionTable
	signatures   *WasmSignatureTable
	types        map[string]WasmType
	variables    map[*ast.Object]WasmVariable
	imports      map[string]*WasmImport
	assertReturn []string
	invoke       []string
	memory       *WasmMemory
	freePtrAddr  int32
}

// For function types
type WasmSignatureTable struct {
	signatures map[*WasmTypeFunc]string
}

// For indirect calls
type WasmFunctionTable struct {
	funcIndex map[*WasmFunc]int
}

type WasmGoSourceFile struct {
	astFile *ast.File
	fset    *token.FileSet
	module  *WasmModule
	pkgName string
	imports map[string]string
}

func NewWasmModuleLinker() WasmModuleLinker {
	sigTable := &WasmSignatureTable{
		signatures: make(map[*WasmTypeFunc]string),
	}
	fnPtrTable := &WasmFunctionTable{
		funcIndex: make(map[*WasmFunc]int),
	}
	m := &WasmModule{
		indent:       0,
		files:        make([]*WasmGoSourceFile, 0, 10),
		functions:    make([]*WasmFunc, 0, 10),
		functionMap:  make(map[*ast.FuncDecl]*WasmFunc),
		functionMap2: make(map[*ast.Object]*WasmFunc),
		funcSymTab:   make(map[string]*WasmFunc),
		funcPtrTable: fnPtrTable,
		signatures:   sigTable,
		types:        make(map[string]WasmType),
		variables:    make(map[*ast.Object]WasmVariable),
		imports:      make(map[string]*WasmImport),
		assertReturn: make([]string, 0, 10),
		invoke:       make([]string, 0, 10),
		memory:       createMemory(1024),
	}
	return m
}

func (m *WasmModule) addAstFile(f *ast.File, fset *token.FileSet) error {
	file := &WasmGoSourceFile{
		astFile: f,
		fset:    fset,
		module:  m,
		imports: make(map[string]string),
	}
	file.setPackageName()
	m.files = append(m.files, file)
	if ident := f.Name; ident != nil {
		m.name = ident.Name
		m.namePos = ident.NamePos
	}

	fmt.Printf("Creating symbol tables for '%s'...\n", file.pkgName)
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		default:
			return fmt.Errorf("unimplemented declaration type: %v at %s", decl, positionString(decl.Pos(), file.fset))
		case *ast.FuncDecl:
			fn, err := file.parseAstFuncDeclPass1(decl, fset, m.indent+1)
			if err != nil {
				return err
			}
			m.functions = append(m.functions, fn)
			m.functionMap[decl] = fn
			m.functionMap2[decl.Name.Obj] = fn
			m.funcSymTab[fn.name] = fn
		case *ast.GenDecl:
			switch decl.Tok {
			default:
				fmt.Printf("Ignoring GenDecl, token: %v\n", decl.Tok)
			case token.IMPORT:
				err := file.parseAstImportDecl(decl)
				if err != nil {
					return err
				}
			case token.TYPE:
				_, err := file.parseAstTypeDecl(decl)
				if err != nil {
					return err
				}
			case token.VAR:
				_, err := file.parseAstVarDeclGlobal(decl, file.fset)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m *WasmModule) finalize() error {
	for _, file := range m.files {
		fmt.Printf("Finalizing '%s'...\n", file.pkgName)
		err := file.generateCode()
		if err != nil {
			return fmt.Errorf("error in finalizing file %s: %v", file.pkgName, err)
		}
	}
	m.memory.writeInt32(int(m.freePtrAddr), int32(len(m.memory.content)))
	return nil
}

func (file *WasmGoSourceFile) generateCode() error {
	for _, decl := range file.astFile.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			fn, ok := file.module.functionMap[decl]
			if !ok {
				return fmt.Errorf("couldn't find function %s in the symbol table", decl.Name.Name)
			}
			_, err := fn.parseAstFuncDecl()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (file *WasmGoSourceFile) parseAstImportDecl(decl *ast.GenDecl) error {
	if len(decl.Specs) != 1 {
		return fmt.Errorf("unsupported import declaration with %d specs", len(decl.Specs))
	}
	switch spec := decl.Specs[0].(type) {
	default:
		return fmt.Errorf("unsupported import declaration with spec: %v at %s", spec, positionString(spec.Pos(), file.fset))
	case *ast.ImportSpec:
		path := strings.Trim(spec.Path.Value, "\"")
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash <= 0 {
			file.imports[path] = path
		} else {
			lastPart := path[lastSlash+1:]
			file.imports[lastPart] = path
		}
		return nil
	}
}

func (file *WasmGoSourceFile) parseComment(c string) {
	pragmaPrefix := "//wasm:"
	if strings.HasPrefix(c, pragmaPrefix) {
		file.parsePragma(strings.TrimPrefix(c, pragmaPrefix))
	}
}

func (file *WasmGoSourceFile) parsePragma(p string) {
	assertReturnPrefix := "assert_return "
	invokePrefix := "invoke "
	if strings.HasPrefix(p, assertReturnPrefix) {
		file.module.assertReturn = append(file.module.assertReturn, strings.TrimPrefix(p, assertReturnPrefix))
	} else if strings.HasPrefix(p, invokePrefix) {
		file.module.invoke = append(file.module.invoke, strings.TrimPrefix(p, invokePrefix))
	}
}

func (m *WasmModule) printGlobalVars(writer FormattingWriter) {
	var headerPrinted bool
	for _, v := range m.variables {
		switch v := v.(type) {
		case *WasmGlobalVar:
			if !headerPrinted {
				writer.Printf("\n")
				writer.PrintfIndent(1, ";; Global variables\n")
				headerPrinted = true
			}
			v.print(writer)
		}
	}
}

func (m *WasmModule) printImports(writer FormattingWriter) {
	writer.Printf("\n")
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
		if isSymbolPublic(f.origName) || f.origName == "main" {
			writer.PrintfIndent(indent, "(export \"%s\" %s)\n", f.origName, f.name)
		}
	}
}

func (file *WasmGoSourceFile) print(writer FormattingWriter) {
	writer.PrintfIndent(1, ";; File %s\n", file.pkgName)
}

func (m *WasmModule) print(writer FormattingWriter) {
	writer.Printf("(module\n")
	bodyIndent := m.indent + 1
	writer.PrintfIndent(bodyIndent, ";; Go package '%s'\n", m.name)
	for _, f := range m.files {
		f.print(writer)
	}
	m.memory.print(writer)
	m.printGlobalVars(writer)
	m.signatures.print(writer)
	m.funcPtrTable.print(writer)
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

func mangleFunctionName(pkg, fn string) string {
	return astNameToWASM(pkg+"/"+fn, nil)
}

func positionString(pos token.Pos, fset *token.FileSet) string {
	position := fset.File(pos).PositionFor(pos, false)
	return fmt.Sprintf("[%v]", position)
}

func (file *WasmGoSourceFile) setPackageName() {
	pos := file.astFile.Package
	position := file.fset.File(pos).PositionFor(pos, false)
	path := position.Filename
	lastSlash := strings.LastIndex(path, "/")
	path = path[:lastSlash]
	// TODO: support other path patterns.
	if strings.HasPrefix(path, "src/") {
		path = path[4:]
	}
	file.pkgName = path
}

func (file *WasmGoSourceFile) getSingleLineGoSource(node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, file.fset, node)
	s := buf.String()
	if strings.Contains(s, "\n") {
		return ""
	} else {
		return s
	}
}

func (e *GoWasmError) Error() string {
	return e.msg
}

func (file *WasmGoSourceFile) ErrorNode(node ast.Node, format string, a ...interface{}) error {
	pos := node.Pos()
	position := file.fset.File(pos).PositionFor(pos, false)
	s := fmt.Sprintf(format, a...)
	src := file.getSingleLineGoSource(node)
	if src != "" {
		s = fmt.Sprintf("%s (src: %s)", s, src)
	}
	e := &GoWasmError{
		msg: fmt.Sprintf("%s @ %v", s, position),
	}
	return e
}

func (tab *WasmFunctionTable) print(writer FormattingWriter) {
	length := len(tab.funcIndex)
	if length > 0 {
		writer.Printf("\n")
		writer.PrintfIndent(1, "(table\n")
		sorted := make([]*WasmFunc, length, length)
		for fn, i := range tab.funcIndex {
			sorted[i] = fn
		}
		for _, fn := range sorted {
			writer.PrintfIndent(2, "%s\n", fn.name)
		}
		writer.PrintfIndent(1, ") ;;table\n")
	}
}

func (tab *WasmFunctionTable) add(fn *WasmFunc) int {
	idx, ok := tab.funcIndex[fn]
	if !ok {
		idx = len(tab.funcIndex)
		tab.funcIndex[fn] = idx
	}
	return idx
}

func (tab *WasmSignatureTable) print(writer FormattingWriter) {
	if len(tab.signatures) > 0 {
		writer.Printf("\n")
	}
	for sig := range tab.signatures {
		sig.printType(writer)
	}
}

func (tab *WasmSignatureTable) equivalent(a, b *WasmTypeFunc) bool {
	if a.result != b.result {
		return false
	}
	if len(a.params) != len(b.params) {
		return false
	}
	for i, t := range a.params {
		if t != b.params[i] {
			return false
		}
	}
	return true
}

func (tab *WasmSignatureTable) add(ty *WasmTypeFunc) *WasmTypeFunc {
	name, ok := tab.signatures[ty]
	if !ok {
		for t, _ := range tab.signatures {
			if tab.equivalent(ty, t) {
				return t
			}
		}
		name = fmt.Sprintf("$F%d", len(tab.signatures))
		ty.wasmName = name
		tab.signatures[ty] = name
	}
	return ty
}
