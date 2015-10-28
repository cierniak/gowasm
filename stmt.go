package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (f *WasmFunc) parseBody(body *ast.BlockStmt) {
	for _, stmt := range body.List {
		var err error
		var expr WasmExpression
		switch stmt := stmt.(type) {
		default:
			panic(fmt.Errorf("unimplemented statement: %v", stmt))
		case *ast.AssignStmt:
			expr, err = f.parseAssignStmt(stmt)
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

func (f *WasmFunc) parseAssignStmt(stmt *ast.AssignStmt) (WasmExpression, error) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return nil, fmt.Errorf("unimplemented multi-value AssignStmt")
	}
	if stmt.Tok != token.DEFINE {
		// TODO: support other assignments
		return nil, fmt.Errorf("unimplemented AssignStmt, token='%v'", stmt.Tok)
	}
	rhs, err := f.parseExpr(stmt.Rhs[0], nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing RHS of an assignment: %v", err)
	}
	ty := rhs.getType()
	if ty == nil {
		return nil, fmt.Errorf("error parsing RHS of an assignment: type is nil")
	}

	switch lhs := stmt.Lhs[0].(type) {
	default:
		return nil, fmt.Errorf("unimplemented LHS in assignment: %v at %s", lhs, positionString(lhs.Pos(), f.fset))
	case *ast.Ident:
		v := &WasmLocal{
			astIdent: lhs,
			name:     astNameToWASM(lhs.Name),
			t:        ty,
		}
		f.module.variables[lhs.Obj] = v
		f.locals = append(f.locals, v)
		s := &WasmSetLocal{
			lhs:  v,
			rhs:  rhs,
			stmt: stmt,
		}
		return s, nil
	}
}

func (f *WasmFunc) parseExprStmt(stmt *ast.ExprStmt) (WasmExpression, error) {
	expr, err := f.parseExpr(stmt.X, nil)
	if err != nil {
		return nil, fmt.Errorf("unimplemented ExprStmt: %v", err)
	}
	return expr, nil
}

func (f *WasmFunc) parseReturnStmt(stmt *ast.ReturnStmt) (WasmExpression, error) {
	r := &WasmReturn{
		stmt: stmt,
	}
	if stmt.Results != nil {
		if len(stmt.Results) != 1 {
			return nil, fmt.Errorf("unimplemented multi-value return statement")
		}
		value, err := f.parseExpr(stmt.Results[0], f.result.t)
		if err != nil {
			return nil, err
		}
		r.value = value
	}
	return r, nil
}
