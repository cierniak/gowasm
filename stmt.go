package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (f *WasmFunc) parseFuncBody(body *ast.BlockStmt) {
	f.expressions = f.parseBody(body, f.indent+1)
}

func (f *WasmFunc) parseBody(body *ast.BlockStmt, indent int) []WasmExpression {
	expressions := make([]WasmExpression, 0, 10)
	for _, stmt := range body.List {
		var err error
		var expr WasmExpression
		switch stmt := stmt.(type) {
		default:
			panic(fmt.Errorf("unimplemented statement: %v", stmt))
		case *ast.AssignStmt:
			expr, err = f.parseAssignStmt(stmt, indent)
		case *ast.ExprStmt:
			expr, err = f.parseExprStmt(stmt, indent)
		case *ast.IfStmt:
			expr, err = f.parseIfStmt(stmt, indent)
		case *ast.ReturnStmt:
			expr, err = f.parseReturnStmt(stmt, indent)
		}
		if err != nil {
			panic(err)
		}
		if expr != nil {
			expressions = append(expressions, expr)
		}
	}
	return expressions
}

func (f *WasmFunc) parseAssignStmt(stmt *ast.AssignStmt, indent int) (WasmExpression, error) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return nil, fmt.Errorf("unimplemented multi-value AssignStmt")
	}
	if stmt.Tok != token.DEFINE {
		// TODO: support other assignments
		return nil, fmt.Errorf("unimplemented AssignStmt, token='%v'", stmt.Tok)
	}
	rhs, err := f.parseExpr(stmt.Rhs[0], nil, indent+1)
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
		s.setIndent(indent + 1)
		return s, nil
	}
}

func (f *WasmFunc) parseExprStmt(stmt *ast.ExprStmt, indent int) (WasmExpression, error) {
	expr, err := f.parseExpr(stmt.X, nil, indent)
	if err != nil {
		return nil, fmt.Errorf("error in ExprStmt: %v", err)
	}
	return expr, nil
}

func (f *WasmFunc) parseBlockStmt(stmt *ast.BlockStmt, indent int) (*WasmBlock, error) {
	expressions := f.parseBody(stmt, indent+1)
	b := &WasmBlock{
		expressions: expressions,
		stmt:        stmt,
	}
	b.setIndent(indent)
	return b, nil
}

func (f *WasmFunc) parseIfStmt(stmt *ast.IfStmt, indent int) (WasmExpression, error) {
	if stmt.Init != nil {
		return nil, fmt.Errorf("unimplemented IfStmt with an init")
	}
	if stmt.Else != nil {
		return nil, fmt.Errorf("unimplemented IfStmt with an else")
	}
	cond, err := f.parseExpr(stmt.Cond, nil, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error in condition of an IfStmt: %v", err)
	}
	body, err := f.parseBlockStmt(stmt.Body, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error in the block of an IfStmt: %v", err)
	}
	i := &WasmIf{
		cond: cond,
		body: body,
		stmt: stmt,
	}
	i.setIndent(indent)
	return i, nil
}

func (f *WasmFunc) parseReturnStmt(stmt *ast.ReturnStmt, indent int) (WasmExpression, error) {
	r := &WasmReturn{
		stmt: stmt,
	}
	r.setIndent(indent)
	if stmt.Results != nil {
		if len(stmt.Results) != 1 {
			return nil, fmt.Errorf("unimplemented multi-value return statement")
		}
		value, err := f.parseExpr(stmt.Results[0], f.result.t, indent+1)
		if err != nil {
			return nil, err
		}
		r.value = value
	}
	return r, nil
}
