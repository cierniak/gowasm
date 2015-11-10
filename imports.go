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
	params     []WasmType
	result     WasmType // TODO(cierniak): multiple values may be returned.
	indent     int
}

// ( call_import <var> <expr>* )
type WasmCallImport struct {
	WasmExprBase
	i    *WasmImport
	args []WasmExpression
	call *ast.CallExpr
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

func (s *WasmScope) parseWASMRuntimeSignature(name string) ([]WasmType, error) {
	result := make([]WasmType, 0, 10)
	parts := strings.Split(name, "_")
	for _, typeName := range parts[1:] {
		t, ok := s.f.module.types[typeName]
		if !ok {
			// TODO(cierniak): Implement this case.
			return nil, fmt.Errorf("type %s hasn't been used yet", typeName)
		}
		result = append(result, t)
	}
	return result, nil
}

func (s *WasmScope) parseWASMRuntimeCall(ident *ast.Ident, call *ast.CallExpr, indent int) (WasmExpression, error) {
	name := ident.Name
	params, err := s.parseWASMRuntimeSignature(name)
	if err != nil {
		return nil, err
	}
	i, ok := s.f.module.imports[name]
	if !ok {
		i = &WasmImport{
			name:       astNameToWASM(name, nil),
			moduleName: "stdio", // TODO(cierniak): support other WASM modules
			funcName:   "print", // TODO(cierniak): support other functions
			params:     params,
			indent:     s.f.module.indent + 1,
		}
		s.f.module.imports[name] = i
	}
	args := s.parseArgs(call.Args, indent+1)
	c := &WasmCallImport{
		i:    i,
		args: args,
		call: call,
	}
	c.setIndent(indent)
	return c, nil
}

func (p *WasmCallImport) getType() WasmType {
	// TODO
	return nil
}

func (c *WasmCallImport) print(writer FormattingWriter) {
	writer.PrintfIndent(c.getIndent(), "(call_import %s\n", c.i.name)
	for _, arg := range c.args {
		arg.print(writer)
	}
	writer.PrintfIndent(c.getIndent(), ")\n")
}

func (c *WasmCallImport) getNode() ast.Node {
	if c.call == nil {
		return nil
	} else {
		return c.call
	}
}
