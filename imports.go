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
	"wasm.Print_int32": {"spectest", "print", "int32->"},
	"wasm.Print_int64": {"spectest", "print", "int64->"},
	"v8.Puts":          {"", "puts", "int32->int32"},
}

func (i *WasmImport) print(writer FormattingWriter) {
	writer.PrintfIndent(i.indent, "(import %s \"%s\" \"%s\" (param", i.name, i.moduleName, i.funcName)
	for _, p := range i.params {
		writer.Printf(" ")
		p.print(writer)
	}
	writer.Printf(")")
	if i.result != nil {
		writer.Printf(" (result ")
		i.result.print(writer)
		writer.Printf(")")
	}
	writer.Printf(")\n")
}

func (s *WasmScope) parseWASMRuntimeSignature(sig string) ([]WasmType, WasmType, error) {
	partsTopLevel := strings.Split(sig, "->")
	if len(partsTopLevel) != 2 {
		return nil, nil, fmt.Errorf("error separating param and return types in an import: %s", sig)
	}
	params := partsTopLevel[0]
	result := make([]WasmType, 0, 10)
	parts := strings.Split(params, ",")
	for _, typeName := range parts {
		t, ok := s.f.module.types[typeName]
		if !ok {
			// TODO(cierniak): Implement this case.
			return nil, nil, fmt.Errorf("param type %s hasn't been used yet", typeName)
		}
		result = append(result, t)
	}

	ret := partsTopLevel[1]
	var retType WasmType
	if ret != "" {
		var ok bool
		retType, ok = s.f.module.types[ret]
		if !ok {
			// TODO(cierniak): Implement this case.
			return nil, nil, fmt.Errorf("return type %s hasn't been used yet", ret)
		}
	}

	return result, retType, nil
}

func (s *WasmScope) parseWASMRuntimeCall(pkg string, ident *ast.Ident, call *ast.CallExpr, indent int) (WasmExpression, error) {
	name := ident.Name
	namesWASM, ok := functionNames[pkg+"."+name]
	if !ok {
		return nil, fmt.Errorf("couldn't find name mapping for import %s", name)
	}
	params, result, err := s.parseWASMRuntimeSignature(namesWASM.signature)
	if err != nil {
		return nil, err
	}
	i, ok := s.f.module.imports[name]
	if !ok {
		i = &WasmImport{
			name:       astNameToWASM(name, nil),
			moduleName: namesWASM.module,
			funcName:   namesWASM.function,
			params:     params,
			result:     result,
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

func (c *WasmCallImport) getType() WasmType {
	return c.i.result
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
