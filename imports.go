package main

import (
	"fmt"
	"go/ast"
	"strings"
)

// import:  ( import <name>? "<module_name>" "<func_name>" (param <type>* ) (result <type>)* )
type WasmImport struct {
	name       string
	moduleName string
	funcName   string
	params     []*WasmType
	result     *WasmType // TODO(cierniak): syntax implies that multiple values may be returned.
	indent     int
}

// ( call_import <var> <expr>* )
type WasmCallImport struct {
	i    *WasmImport
	args []WasmExpression
}

func (i *WasmImport) print(writer FormattingWriter) {
	writer.PrintfIndent(i.indent, "(import %s \"%s\" \"%s\" (param", i.name, i.moduleName, i.funcName)
	for _, p := range i.params {
		writer.Printf(" ")
		p.print(writer)
	}
	writer.Printf("))\n")
}

func isWASMRuntimePackage(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "wasm"
}

func (f *WasmFunc) parseWASMRuntimeSignature(name string) ([]*WasmType, error) {
	result := make([]*WasmType, 0, 10)
	parts := strings.Split(name, "_")
	for _, typeName := range parts[1:] {
		t, ok := f.module.types[typeName]
		if !ok {
			// TODO(cierniak): Implement this case.
			return nil, fmt.Errorf("type %s hasn't been used yet", typeName)
		}
		result = append(result, t)
	}
	return result, nil
}

func (f *WasmFunc) parseWASMRuntimeCall(ident *ast.Ident, call *ast.CallExpr) (WasmExpression, error) {
	name := ident.Name
	params, err := f.parseWASMRuntimeSignature(name)
	if err != nil {
		return nil, err
	}
	i, ok := f.module.imports[name]
	if !ok {
		i = &WasmImport{
			name:       astNameToWASM(name),
			moduleName: "stdio", // TODO(cierniak): support other WASM modules
			funcName:   "print", // TODO(cierniak): support other functions
			params:     params,
			indent:     f.module.indent + 1,
		}
		f.module.imports[name] = i
	}
	args := f.parseArgs(call.Args)
	c := &WasmCallImport{
		i:    i,
		args: args,
	}
	return c, nil
}

func (p *WasmCallImport) getType() *WasmType {
	// TODO
	return nil
}

func (c *WasmCallImport) print(writer FormattingWriter) {
	writer.Printf("(call_import %s", c.i.name)
	for _, arg := range c.args {
		writer.Printf(" ")
		arg.print(writer)
	}
	writer.Printf(")")
}
