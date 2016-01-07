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

type ImportName struct {
	module    string
	function  string
	signature string
}

var functionNames map[string]ImportName = map[string]ImportName{
	"Print_int32": {"spectest", "print", "int32->"},
	"Print_int64": {"spectest", "print", "int64->"},
	"Puts":        {"", "puts", "int32->int32"},
}

func (i *WasmImport) print(writer FormattingWriter) {
	writer.PrintfIndent(i.indent, "(import %s \"%s\" \"%s\" (param", i.name, i.moduleName, i.funcName)
	for _, p := range i.params {
		writer.Printf(" ")
		p.print(writer)
	}
	writer.Printf("))\n")
}

func (s *WasmScope) parseWASMRuntimeSignature(sig string) ([]WasmType, error) {
	partsTopLevel := strings.Split(sig, "->")
	if len(partsTopLevel) != 2 {
		return nil, fmt.Errorf("error separating param and return types in an import: %s", sig)
	}
	params := partsTopLevel[0]
	//ret := partsTopLevel[1]
	result := make([]WasmType, 0, 10)
	parts := strings.Split(params, "_")
	for _, typeName := range parts {
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
	namesWASM, ok := functionNames[name]
	if !ok {
		return nil, fmt.Errorf("couldn't find name mapping for import %s", name)
	}
	params, err := s.parseWASMRuntimeSignature(namesWASM.signature)
	if err != nil {
		return nil, err
	}
	fmt.Printf("parseWASMRuntimeCall, name: %s, param: %v\n", name, params)
	i, ok := s.f.module.imports[name]
	if !ok {
		i = &WasmImport{
			name:       astNameToWASM(name, nil),
			moduleName: namesWASM.module,
			funcName:   namesWASM.function,
			params:     params,
			indent:     s.f.module.indent + 1,
		}
		s.f.module.imports[name] = i
	}
	args, err := s.parseArgs(call.Args, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error parsing args to runtime function %s: %v", name, err)
	}
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
