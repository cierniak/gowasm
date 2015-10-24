package main

import (
	"fmt"
	"go/ast"
)

func (f *WasmFunc) parseBody(body *ast.BlockStmt) {
	for _, stmt := range body.List {
		var err error
		var expr WasmExpression
		switch stmt := stmt.(type) {
		default:
			panic(fmt.Errorf("unimplemented statement: %v", stmt))
		case *ast.ExprStmt:
			expr, err = f.parseExprStmt(stmt)
		case *ast.ReturnStmt:
			expr, err = f.parseReturnStmt(stmt)
		}
		if err != nil {
			panic(err)
		}
		if expr != nil {
			f.expressions = append(f.expressions, expr)
		}
	}
}

func (f *WasmFunc) parseExprStmt(stmt *ast.ExprStmt) (WasmExpression, error) {
	expr, err := f.parseExpr(stmt.X)
	if err != nil {
		return nil, fmt.Errorf("unimplemented ExprStmt: %v", err)
	}
	return expr, nil
}

func (f *WasmFunc) parseReturnStmt(stmt *ast.ReturnStmt) (WasmExpression, error) {
	r := &WasmReturn{}
	if stmt.Results != nil {
		if len(stmt.Results) != 1 {
			return nil, fmt.Errorf("unimplemented multi-value return statement")
		}
		value, err := f.parseExpr(stmt.Results[0])
		if err != nil {
			return nil, err
		}
		r.value = value
	}
	return r, nil
}
