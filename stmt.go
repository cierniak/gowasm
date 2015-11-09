package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

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

func (f *WasmFunc) createScope(indent int) *WasmScope {
	s := &WasmScope{
		f:           f,
		indent:      indent,
		expressions: make([]WasmExpression, 0, 10),
		n:           f.nextScope,
	}
	s.name = fmt.Sprintf("scope%d", s.n)
	f.nextScope++
	return s
}

func (f *WasmFunc) parseStmtList(stmts []ast.Stmt, indent int, optionalScope *WasmScope) (*WasmScope, error) {
	scope := optionalScope
	if scope == nil {
		scope = f.createScope(indent)
	}
	err := scope.parseStmtList(stmts, indent)
	return scope, err
}

func (s *WasmScope) parseStmt(stmt ast.Stmt, indent int) (WasmExpression, error) {
	switch stmt := stmt.(type) {
	default:
		panic(fmt.Errorf("unimplemented statement: %v", stmt))
	case *ast.AssignStmt:
		return s.parseAssignStmt(stmt, indent)
	case *ast.ExprStmt:
		return s.parseExprStmt(stmt, indent)
	case *ast.ForStmt:
		return s.parseForStmt(stmt, indent)
	case *ast.IfStmt:
		return s.parseIfStmt(stmt, indent)
	case *ast.IncDecStmt:
		return s.parseIncDecStmt(stmt, indent)
	case *ast.ReturnStmt:
		return s.parseReturnStmt(stmt, indent)
	}
}

func (s *WasmScope) createNop(indent int) *WasmNop {
	n := &WasmNop{}
	n.setIndent(indent)
	return n
}

func (s *WasmScope) parseDefineAssignLHS(lhs []ast.Expr, ty *WasmType, indent int) (WasmVariable, error) {
	if len(lhs) != 1 {
		return nil, fmt.Errorf("unimplemented multi-value LHS in AssignStmt")
	}
	switch lhs := lhs[0].(type) {
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
		return v, nil
	}
}

func (s *WasmScope) parseAssignLHS(lhs []ast.Expr, ty *WasmType, indent int) (WasmVariable, error) {
	if len(lhs) != 1 {
		return nil, fmt.Errorf("unimplemented multi-value LHS in AssignStmt")
	}
	switch lhs := lhs[0].(type) {
	default:
		return nil, fmt.Errorf("unimplemented LHS in assignment: %v at %s", lhs, positionString(lhs.Pos(), s.f.fset))
	case *ast.Ident:
		v, ok := s.f.module.variables[lhs.Obj]
		if !ok {
			return nil, fmt.Errorf("couldn't find variable '%s' on the LHS of an assignment", lhs.Name)
		}
		return v, nil
	}
}

func (s *WasmScope) parseAssignStmt(stmt *ast.AssignStmt, indent int) (WasmExpression, error) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return nil, fmt.Errorf("unimplemented multi-value AssignStmt")
	}

	var err error
	rhs, err := s.f.parseExpr(stmt.Rhs[0], nil, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error parsing RHS of an assignment: %v", err)
	}
	ty := rhs.getType()
	if ty == nil {
		return nil, fmt.Errorf("error parsing RHS of an assignment: type is nil")
	}

	var v WasmVariable
	switch stmt.Tok {
	default:
		return nil, fmt.Errorf("unimplemented AssignStmt, token='%v'", stmt.Tok)
	case token.ASSIGN:
		v, err = s.parseAssignLHS(stmt.Lhs, ty, indent)
	case token.DEFINE:
		v, err = s.parseDefineAssignLHS(stmt.Lhs, ty, indent)
	}
	if err != nil {
		return nil, err
	}
	sl := &WasmSetLocal{
		lhs:  v,
		rhs:  rhs,
		stmt: stmt,
	}
	sl.setIndent(indent)
	return sl, nil
}

func (s *WasmScope) parseExprStmt(stmt *ast.ExprStmt, indent int) (WasmExpression, error) {
	expr, err := s.f.parseExpr(stmt.X, nil, indent)
	if err != nil {
		return nil, fmt.Errorf("error in ExprStmt: %v", err)
	}
	return expr, nil
}

func (s *WasmScope) parseForStmt(stmt *ast.ForStmt, indent int) (WasmExpression, error) {
	var outerScope *WasmScope
	var err error
	if stmt.Init != nil {
		init := []ast.Stmt{stmt.Init}
		outerScope, err = s.f.parseStmtList(init, indent+1, nil)
		if err != nil {
			return nil, fmt.Errorf("error in the init part of a loop: %v", err)
		}
	}

	cond, err := s.f.parseExpr(stmt.Cond, nil, indent+3)
	if err != nil {
		return nil, fmt.Errorf("error in the condition of a loop: %v", err)
	}
	b := &WasmBreak{}
	b.setIndent(indent + 3)
	ifStmt, err := s.createIf(cond, s.createNop(indent+3), b, indent+2)
	if err != nil {
		return nil, fmt.Errorf("error in the condition stmt of a loop: %v", err)
	}
	scope := s.f.createScope(indent)
	scope.expressions = append(scope.expressions, ifStmt)

	scope, err = s.f.parseStmtList(stmt.Body.List, indent+2, scope)
	if err != nil {
		return nil, fmt.Errorf("error in the body of a loop: %v", err)
	}

	if stmt.Post != nil {
		post := []ast.Stmt{stmt.Post}
		scope, err = s.f.parseStmtList(post, indent+2, scope)
		if err != nil {
			return nil, fmt.Errorf("error in the post part of a loop: %v", err)
		}
	}

	l := &WasmLoop{
		stmt:  stmt,
		scope: scope,
	}
	l.setIndent(indent + 1)

	if outerScope == nil {
		return nil, fmt.Errorf("loops with no init are not implemented")
	}
	outerScope.expressions = append(outerScope.expressions, l)
	outerBlock := s.createBlock(outerScope, stmt, indent)

	return outerBlock, nil
}

func (s *WasmScope) createBlock(scope *WasmScope, stmt ast.Stmt, indent int) *WasmBlock {
	b := &WasmBlock{
		scope: scope,
		stmt:  stmt,
	}
	b.setIndent(indent)
	return b
}

func (s *WasmScope) parseBlockStmt(stmt *ast.BlockStmt, indent int) (*WasmBlock, error) {
	scope, err := s.f.parseStmtList(stmt.List, indent+1, nil)
	if err != nil {
		return nil, err
	}
	return s.createBlock(scope, stmt, indent), nil
}

func (s *WasmScope) createIf(cond, body, bodyElse WasmExpression, indent int) (*WasmIf, error) {
	i := &WasmIf{
		cond:     cond,
		body:     body,
		bodyElse: bodyElse,
	}
	i.setIndent(indent)
	return i, nil
}

func (s *WasmScope) parseIfStmt(stmt *ast.IfStmt, indent int) (WasmExpression, error) {
	if stmt.Init != nil {
		return nil, fmt.Errorf("unimplemented IfStmt with an init")
	}
	if stmt.Else != nil {
		return nil, fmt.Errorf("unimplemented IfStmt with an else")
	}
	cond, err := s.f.parseExpr(stmt.Cond, nil, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error in condition of an IfStmt: %v", err)
	}
	body, err := s.parseBlockStmt(stmt.Body, indent+1)
	if err != nil {
		return nil, fmt.Errorf("error in the block of an IfStmt: %v", err)
	}
	i, err := s.createIf(cond, body, nil, indent)
	if err != nil {
		return nil, fmt.Errorf("error creating an IfStmt: %v", err)
	}
	i.stmt = stmt
	return i, nil
}

func (s *WasmScope) parseIncDecStmt(stmt *ast.IncDecStmt, indent int) (WasmExpression, error) {
	switch x := stmt.X.(type) {
	default:
		return nil, fmt.Errorf("unimplemented expr in IncDecStmt: %v at %s", x, positionString(x.Pos(), s.f.fset))
	case *ast.Ident:
		v, ok := s.f.module.variables[x.Obj]
		if !ok {
			return nil, fmt.Errorf("undefined variable '%s' in IncDecStmt at %s", x.Obj.Name, positionString(x.Pos(), s.f.fset))
		}
		vRHS, err := s.f.parseIdent(x, indent+2)
		if err != nil {
			return nil, fmt.Errorf("error in IncDecStmt: %v", err)
		}

		inc, err := s.f.createLiteral("1", v.getType(), indent+2)
		if err != nil {
			return nil, fmt.Errorf("error in IncDecStmt: %v", err)
		}

		rhs, err := s.f.createBinaryExpr(vRHS, inc, binOpMapping[stmt.Tok], v.getType(), indent+1)
		if err != nil {
			return nil, fmt.Errorf("error in IncDecStmt: %v", err)
		}

		sl := &WasmSetLocal{
			lhs:  v,
			rhs:  rhs,
			stmt: stmt,
		}
		sl.setIndent(indent)
		return sl, nil
	}
	return nil, fmt.Errorf("not implemented: IncDecStmt")
}

func (s *WasmScope) parseReturnStmt(stmt *ast.ReturnStmt, indent int) (WasmExpression, error) {
	r := &WasmReturn{
		stmt: stmt,
	}
	r.setIndent(indent)
	if stmt.Results != nil {
		if len(stmt.Results) != 1 {
			return nil, fmt.Errorf("unimplemented multi-value return statement")
		}
		value, err := s.f.parseExpr(stmt.Results[0], s.f.result.t, indent+1)
		if err != nil {
			return nil, err
		}
		r.value = value
	}
	return r, nil
}
