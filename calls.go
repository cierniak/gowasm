package main

import (
	"fmt"
	"go/ast"
)

type WasmFuncPtr struct {
	WasmExprBase
	def *WasmFunc
	idx WasmExpression
}

type WasmCallBase struct {
	WasmExprBase
	args []WasmExpression
	call *ast.CallExpr
}

// ( call <var> <expr>* )
type WasmCall struct {
	WasmCallBase
	name string
	def  *WasmFunc
}

// ( call_indirect <var> <expr> <expr>* )
type WasmCallIndirect struct {
	WasmCallBase
	name      string
	signature *WasmTypeFunc
	index     WasmExpression
}

func (s *WasmScope) parseArgs(args []ast.Expr, indent int) ([]WasmExpression, error) {
	result := make([]WasmExpression, 0, len(args))
	for i, arg := range args {
		e, err := s.parseExpr(arg, nil, indent) // TODO: Should the hint be nil?
		if err != nil {
			return nil, fmt.Errorf("couldn't parse arg #%d: %v", i, err)
		}
		result = append(result, e)
	}
	return result, nil
}

func (s *WasmScope) createCallExprWithArgs(call *ast.CallExpr, name string, fn *WasmFunc, args []WasmExpression, indent int) (WasmExpression, error) {
	c := &WasmCall{
		name: name,
		def:  fn,
	}
	c.args = args
	c.call = call
	c.setIndent(indent)
	c.setNode(call)
	c.setScope(s)
	if fn.result != nil {
		c.setFullType(fn.result.t)
	}
	return c, nil
}

func (s *WasmScope) createCallExpr(call *ast.CallExpr, name string, fn *WasmFunc, indent int) (WasmExpression, error) {
	args, err := s.parseArgs(call.Args, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error parsing args to function %s: %v", name, err)
	}
	return s.createCallExprWithArgs(call, name, fn, args, indent)
}

func (s *WasmScope) createIndirectCallExpr(call *ast.CallExpr, name string, ident *ast.Ident, indent int) (WasmExpression, error) {
	idx, err := s.parseIdent(ident, indent+1)
	if err != nil {
		return nil, fmt.Errorf("call_indirect, couldn't create expression for the table index")
	}
	c := &WasmCallIndirect{
		name:  name,
		index: idx,
	}
	c.call = call
	c.setIndent(indent)
	c.setNode(call)
	c.setScope(s)
	switch ty := idx.getType().(type) {
	default:
		return nil, s.f.file.ErrorNode(call, "unimplemented expression type: %v", ty)
	case *WasmTypeFunc:
		fmt.Printf("createIndirectCallExpr, name: %s\n", ty.wasmName)
		c.signature = ty
	}
	return c, nil
}

func (s *WasmScope) parseCallExpr(call *ast.CallExpr, indent int) (WasmExpression, error) {
	switch fun := call.Fun.(type) {
	default:
		return nil, s.f.file.ErrorNode(call, "unimplemented function")
	case *ast.Ident:
		typ, err := s.f.file.parseAstType(fun)
		if err == nil && len(call.Args) == 1 {
			return s.parseConvertExpr(typ, call.Args[0], indent)
		}

		var name string
		fn, ok := s.f.module.functionMap2[fun.Obj]
		if ok {
			name = fn.name
		} else {
			name = astNameToWASM(fun.Name, nil)
		}

		// TODO: Make it work for forward references.
		decl, ok := fun.Obj.Decl.(*ast.FuncDecl)
		if ok {
			return s.createCallExpr(call, name, s.f.module.functionMap[decl], indent)
		} else {
			_, ok := s.f.module.variables[fun.Obj]
			if !ok {
				return nil, fmt.Errorf("function %s undefined (forward reference?)", name)
			}
			return s.createIndirectCallExpr(call, name, fun, indent)
		}
	case *ast.ParenExpr:
		typ, err := s.f.file.parseAstType(fun.X)
		if err == nil && len(call.Args) == 1 {
			return s.parseConvertExpr(typ, call.Args[0], indent)
		}
		return nil, s.f.file.ErrorNode(call, "unimplemented function: ParenExpr")
	case *ast.SelectorExpr:
		return s.parseCallExprSelector(call, fun, indent)
	}
	return nil, fmt.Errorf("unimplemented call expression at %s", positionString(call.Lparen, s.f.fset))
}

func (s *WasmScope) parseUnsafePkgCall(ident *ast.Ident, call *ast.CallExpr, indent int) (WasmExpression, error) {
	name := ident.Name
	switch name {
	default:
		return nil, s.f.file.ErrorNode(call, "member of 'package unsafe' is not implemented yet: %s", name)
	case "Pointer":
		t, err := s.f.file.module.convertAstTypeNameToWasmType("unsafe.Pointer")
		if err != nil {
			return nil, s.f.file.ErrorNode(call, "%v", err)
		}
		if len(call.Args) == 1 {
			return s.parseConvertExpr(t, call.Args[0], indent)
		} else {
			return nil, s.f.file.ErrorNode(call, "unexpected number of arguments to unsafe.Pointer")
		}
	}
}

func (s *WasmScope) parseCallExprSelector(call *ast.CallExpr, se *ast.SelectorExpr, indent int) (WasmExpression, error) {
	switch x := se.X.(type) {
	default:
		return nil, fmt.Errorf("unimplemented X in selector: %v", x)
	case *ast.Ident:
		pkgShort := x.Name
		switch pkgShort {
		case "unsafe":
			return s.parseUnsafePkgCall(se.Sel, call, indent)
		case "wasm":
			return s.parseWASMRuntimeCall(se.Sel, call, indent)
		}
		pkgLong, ok := s.f.file.imports[pkgShort]
		if ok {
			name := mangleFunctionName(pkgLong, se.Sel.Name)
			fn, ok := s.f.module.funcSymTab[name]
			if !ok {
				return nil, fmt.Errorf("link error, couldn't find function: %s", name)
			}
			return s.createCallExpr(call, name, fn, indent)
		}
	}
	return nil, fmt.Errorf("unimplemented selector in a call expression, X: %v, sel: %v", se.X, se.Sel)
}

func (s *WasmScope) parseFuncIdent(ident *ast.Ident, fn *WasmFunc, indent int) (WasmExpression, error) {
	idx, err := s.createLiteralInt32(int32(fn.tabIndex), indent+1)
	if err != nil {
		return nil, fmt.Errorf("error creating table index: %v", err)
	}
	idx.setComment(fmt.Sprintf("function index for %s", fn.name))
	idx.setNode(ident)
	idx.setScope(s)
	fptr := &WasmFuncPtr{
		def: fn,
		idx: idx,
	}
	fptr.setNode(ident)
	fptr.setScope(s)
	fn.prepareForIndirectCall()
	return fptr, nil
}

func (f *WasmFuncPtr) getType() WasmType {
	return f.def.signature
}

func (f *WasmFuncPtr) print(writer FormattingWriter) {
	f.idx.print(writer)
}

func (c *WasmCall) getType() WasmType {
	if c.def != nil {
		return c.def.result.t
	}
	return nil
}

func (c *WasmCall) print(writer FormattingWriter) {
	writer.PrintfIndent(c.getIndent(), "(call %s%s\n", c.name, c.getComment())
	for _, arg := range c.args {
		arg.print(writer)
	}
	writer.PrintfIndent(c.getIndent(), ") ;; call %s\n", c.name)
}

func (c *WasmCallIndirect) getType() WasmType {
	return c.signature
}

func (c *WasmCallIndirect) print(writer FormattingWriter) {
	writer.PrintfIndent(c.getIndent(), "(call_indirect %s%s\n", c.signature.wasmName, c.getComment())
	c.index.print(writer)
	for _, arg := range c.args {
		arg.print(writer)
	}
	writer.PrintfIndent(c.getIndent(), ") ;; call_indirect %s\n", c.name)
}

func (c *WasmCall) getNode() ast.Node {
	return c.call
}
