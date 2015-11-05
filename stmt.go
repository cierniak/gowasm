package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (f *WasmFunc) parseFuncBody(body *ast.BlockStmt) {
	scope, err := f.parseStmtList(body.List, f.indent+1, nil)
	if err != nil {
		panic(err)
	}
	f.scope = scope
}

func (s *WasmScope) parseStmtList(stmts []ast.Stmt, indent int) error {
	for _, stmt := range stmts {
		expr, err := s.parseStmt(stmt, indent)
		if err != nil {
			return err
		}
		s.expressions = append(s.expressions, expr)
	}
	return nil
}

func (f *WasmFunc) parseStmtList(stmts []ast.Stmt, indent int, optionalScope *WasmScope) (*WasmScope, error) {
	s := optionalScope
	if s == nil {
		s = &WasmScope{
			f:           f,
			indent:      indent,
			expressions: make([]WasmExpression, 0, 10),
			n:           f.nextScope,
		}
		s.name = fmt.Sprintf("scope%d", s.n)
		f.nextScope++
	}
	err := s.parseStmtList(stmts, indent)
	return s, err
}

func (s *WasmScope) parseStmt(stmt ast.Stmt, indent int) (WasmExpression, error) {
	switch stmt := stmt.(type) {
	default:
		panic(fmt.Errorf("unimplemented statement: %v", stmt))
	case *ast.AssignStmt:
		return s.parseAssignStmt(stmt, indent)
	case *ast.ExprStmt:
		return s.f.parseExprStmt(stmt, indent)
	case *ast.ForStmt:
		return s.f.parseForStmt(stmt, indent)
	case *ast.IfStmt:
		return s.f.parseIfStmt(stmt, indent)
	case *ast.ReturnStmt:
		return s.f.parseReturnStmt(stmt, indent)
	}
}

func (s *WasmScope) parseAssignStmt(stmt *ast.AssignStmt, indent int) (WasmExpression, error) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return nil, fmt.Errorf("unimplemented multi-value AssignStmt")
	}
	if stmt.Tok != token.DEFINE {
		// TODO: support other assignments
		return nil, fmt.Errorf("unimplemented AssignStmt, token='%v'", stmt.Tok)
	}
	rhs, err := s.f.parseExpr(stmt.Rhs[0], nil, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error parsing RHS of an assignment: %v", err)
	}
	ty := rhs.getType()
	if ty == nil {
		return nil, fmt.Errorf("error parsing RHS of an assignment: type is nil")
	}

	switch lhs := stmt.Lhs[0].(type) {
	default:
		return nil, fmt.Errorf("unimplemented LHS in assignment: %v at %s", lhs, positionString(lhs.Pos(), s.f.fset))
	case *ast.Ident:
		v := &WasmLocal{
			astIdent: lhs,
			name:     astNameToWASM(lhs.Name, s),
			t:        ty,
		}
		s.f.module.variables[lhs.Obj] = v
		s.f.locals = append(s.f.locals, v)
		sl := &WasmSetLocal{
			lhs:  v,
			rhs:  rhs,
			stmt: stmt,
		}
		sl.setIndent(indent)
		return sl, nil
	}
}

func (f *WasmFunc) parseExprStmt(stmt *ast.ExprStmt, indent int) (WasmExpression, error) {
	expr, err := f.parseExpr(stmt.X, nil, indent)
	if err != nil {
		return nil, fmt.Errorf("error in ExprStmt: %v", err)
	}
	return expr, nil
}

func (f *WasmFunc) parseForStmt(stmt *ast.ForStmt, indent int) (WasmExpression, error) {
	var scope *WasmScope
	var err error
	if stmt.Init != nil {
		init := []ast.Stmt{stmt.Init}
		scope, err = f.parseStmtList(init, indent+1, nil)
		if err != nil {
			return nil, fmt.Errorf("error in the init part of a loop: %v", err)
		}
	}

	scope, err = f.parseStmtList(stmt.Body.List, indent+1, scope)
	if err != nil {
		return nil, fmt.Errorf("error in the body of a loop: %v", err)
	}

	b := &WasmBreak{}
	b.setIndent(indent + 1)
	scope.expressions = append(scope.expressions, b)

	l := &WasmLoop{
		stmt:  stmt,
		scope: scope,
	}
	l.setIndent(indent)
	// TODO: Finish implementing the loop.
	return l, nil
}

func (f *WasmFunc) parseBlockStmt(stmt *ast.BlockStmt, indent int) (*WasmBlock, error) {
	scope, err := f.parseStmtList(stmt.List, f.indent+1, nil)
	if err != nil {
		return nil, err
	}
	b := &WasmBlock{
		scope: scope,
		stmt:  stmt,
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
