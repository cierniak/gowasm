package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

// func:   ( func <name>? <type>? <param>* <result>? <local>* <expr>* )
type WasmFunc struct {
	funcDecl  *ast.FuncDecl
	fset      *token.FileSet
	module    *WasmModule
	file      *WasmGoSourceFile
	indent    int
	name      string
	origName  string
	namePos   token.Pos
	params    []*WasmParam
	result    *WasmResult
	locals    []*WasmLocal
	scope     *WasmScope
	nextScope int
}

// param:  ( param <type>* ) | ( param <name> <type> )
type WasmParam struct {
	astIdent *ast.Ident
	astType  ast.Expr
	name     string
	t        WasmType
	fullType WasmType
}

type WasmLocal struct {
	astIdent *ast.Ident
	name     string
	t        WasmType
	fullType WasmType
}

// result: ( result <type> )
type WasmResult struct {
	astIdent *ast.Ident
	astType  ast.Expr
	name     string
	t        WasmType
}

func (file *WasmGoSourceFile) parseAstFuncDeclPass1(funcDecl *ast.FuncDecl, fset *token.FileSet, indent int) (*WasmFunc, error) {
	f := &WasmFunc{
		funcDecl: funcDecl,
		fset:     fset,
		module:   file.module,
		file:     file,
		indent:   indent,
		params:   make([]*WasmParam, 0, 10),
		locals:   make([]*WasmLocal, 0, 10),
	}
	if ident := funcDecl.Name; ident != nil {
		f.name = mangleFunctionName(file.pkgName, ident.Name)
		f.origName = ident.Name
		f.namePos = ident.NamePos
	}
	if funcDecl.Type != nil {
		f.parseType(funcDecl.Type)
	}
	f.scope = f.createScope(fmt.Sprintf("function_%s", f.origName))
	return f, nil
}

func (f *WasmFunc) parseAstFuncDecl() (*WasmFunc, error) {
	funcDecl := f.funcDecl
	if cg := funcDecl.Doc; cg != nil {
		for _, c := range cg.List {
			f.file.parseComment(c.Text)
		}
	}
	err := f.scope.parseStatementList(funcDecl.Body.List, f.indent+1)
	return f, err
}

func (f *WasmFunc) parseType(t *ast.FuncType) {
	if t.Params.List != nil {
		for _, field := range t.Params.List {
			paramType, err := f.file.parseAstType(field.Type)
			if err != nil {
				panic(err)
			}
			for _, name := range field.Names {
				p := &WasmParam{
					astIdent: name,
					astType:  field.Type,
					name:     astNameToWASM(name.Name, nil),
					t:        paramType,
				}
				f.module.variables[name.Obj] = p
				f.params = append(f.params, p)
			}
		}
	}

	if t.Results != nil {
		if len(t.Results.List) != 1 {
			err := fmt.Errorf("functions returning %d values are not implemented", len(t.Results.List))
			panic(err)
		}
		field := t.Results.List[0]
		paramType, err := f.file.parseAstType(field.Type)
		if err != nil {
			panic(err)
		}
		f.result = &WasmResult{
			astType: field.Type,
			t:       paramType,
		}
	}
}

func (f *WasmFunc) print(writer FormattingWriter) {
	writer.PrintfIndent(f.indent, ";; Go function '%s' %s\n", f.origName, positionString(f.namePos, f.fset))
	writer.PrintfIndent(f.indent, "(func %s", f.name)
	for _, param := range f.params {
		param.print(writer)
	}
	if f.result != nil {
		f.result.print(writer)
	}
	writer.Printf("\n")
	bodyIndent := f.indent + 1
	for _, v := range f.locals {
		writer.PrintfIndent(bodyIndent, "")
		v.print(writer)
	}
	if len(f.locals) > 0 {
		writer.Printf("\n")
	}
	for i, expr := range f.scope.expressions {
		if i > 0 {
			writer.Printf("\n")
		}
		f.printGoSource(bodyIndent, expr.getNode(), writer)
		expr.print(writer)
	}
	writer.PrintfIndent(f.indent, ") ;; func %s\n", f.name)
}

func (f *WasmFunc) getSingleLineGoSource(node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, f.fset, node)
	s := buf.String()
	if strings.Contains(s, "\n") {
		return ""
	} else {
		return s
	}
}

func (f *WasmFunc) printGoSource(bodyIndent int, node ast.Node, writer FormattingWriter) {
	if node == nil || node.Pos() == 0 {
		return
	}
	indentString := strings.Repeat(indentPattern, bodyIndent)
	linePrefix := indentString + ";; "
	var buf bytes.Buffer
	printer.Fprint(&buf, f.fset, node)
	s := buf.String()
	s = strings.Replace(s, "\n", "\n"+linePrefix, -1)
	writer.PrintfIndent(bodyIndent, ";; %s\n", s)
}

func (p *WasmParam) getName() string {
	return p.name
}

func (p *WasmParam) getType() WasmType {
	return p.t
}

func (p *WasmParam) getFullType() WasmType {
	return p.fullType
}

func (p *WasmParam) setFullType(t WasmType) {
	p.fullType = t
}

func (p *WasmParam) print(writer FormattingWriter) {
	writer.Printf(" (param ")
	if p.name != "" {
		writer.Printf("%s ", p.name)
	}
	p.t.print(writer)
	writer.Printf(")")
}

func (r *WasmResult) print(writer FormattingWriter) {
	writer.Printf(" (result ")
	r.t.print(writer)
	writer.Printf(")")
}

func (v *WasmLocal) getName() string {
	return v.name
}

func (v *WasmLocal) getType() WasmType {
	return v.t
}

func (v *WasmLocal) getFullType() WasmType {
	return v.fullType
}

func (v *WasmLocal) setFullType(t WasmType) {
	v.fullType = t
}

func (v *WasmLocal) print(writer FormattingWriter) {
	writer.Printf("(local ")
	if v.name != "" {
		writer.Printf("%s ", v.name)
	}
	v.t.print(writer)
	writer.Printf(") ;; %s\n", v.astIdent.Name)
}
