package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

// Expression list that may introduce new locals, e.g. block or loop.
type WasmScope struct {
	expressions []WasmExpression
	f           *WasmFunc
	n           int
	name        string
}

// ( nop )
type WasmNop struct {
	WasmExprBase
}

// ( block <expr>+ )
// ( block <var> <expr>+ ) ;; = (label <var> (block <expr>+))
type WasmBlock struct {
	WasmExprBase
	scope *WasmScope
	stmt  ast.Stmt
}

// ( return <expr>? )
type WasmReturn struct {
	WasmExprBase
	value WasmExpression
	stmt  *ast.ReturnStmt
}

// ( if <expr> <expr> <expr> )
// ( if <expr> <expr> )
type WasmIf struct {
	WasmExprBase
	cond     WasmExpression
	body     WasmExpression
	bodyElse WasmExpression
	stmt     *ast.IfStmt
}

// ( loop <expr>* ) ;; = (loop (block <expr>*))
// ( loop <var> <var>? <expr>* ) ;; = (label <var> (loop (block <var>? <expr>*)))
type WasmLoop struct {
	WasmExprBase
	scope *WasmScope
	cond  WasmExpression
	body  *WasmBlock
	stmt  *ast.ForStmt
}

// ( break <var> <expr>? )
type WasmBreak struct {
	WasmExprBase
	scope *WasmScope
	v     int
}

// ( set_local <var> <expr> )
type WasmSetLocal struct {
	WasmExprBase
	lhs  WasmVariable
	rhs  WasmExpression
	stmt ast.Stmt
}

func (f *WasmFunc) createScope(prefix string) *WasmScope {
	if prefix == "" {
		prefix = "scope"
	}
	s := &WasmScope{
		f:           f,
		expressions: make([]WasmExpression, 0, 10),
		n:           f.nextScope,
	}
	s.name = fmt.Sprintf("%s%d", prefix, s.n)
	f.nextScope++
	return s
}

func (s *WasmScope) parseStatementList(stmts []ast.Stmt, indent int) error {
	for _, stmt := range stmts {
		expr, err := s.parseStmt(stmt, indent)
		if err != nil {
			return err
		}
		s.expressions = append(s.expressions, expr)
	}
	return nil
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

func (s *WasmScope) parseDefineAssignLHS(lhs []ast.Expr, ty WasmTypeI, indent int) (WasmVariable, error) {
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

func (s *WasmScope) parseAssignLHS(lhs []ast.Expr, ty WasmTypeI, indent int) (WasmVariable, error) {
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
	rhs, err := s.parseExpr(stmt.Rhs[0], nil, indent+1)
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
	expr, err := s.parseExpr(stmt.X, nil, indent)
	if err != nil {
		return nil, fmt.Errorf("error in ExprStmt: %v", err)
	}
	return expr, nil
}

func (s *WasmScope) parseForStmt(stmt *ast.ForStmt, indent int) (WasmExpression, error) {
	outerScope := s.f.createScope("loop_block")
	var err error
	if stmt.Init != nil {
		init := []ast.Stmt{stmt.Init}
		err = outerScope.parseStatementList(init, indent+1)
		if err != nil {
			return nil, fmt.Errorf("error in the init part of a loop: %v", err)
		}
	}

	cond, err := s.parseExpr(stmt.Cond, nil, indent+3)
	if err != nil {
		return nil, fmt.Errorf("error in the condition of a loop: %v", err)
	}
	scope := s.f.createScope("loop")
	b := &WasmBreak{
		scope: scope,
	}
	b.setIndent(indent + 3)
	ifStmt, err := s.createIf(cond, s.createNop(indent+3), b, indent+2)
	if err != nil {
		return nil, fmt.Errorf("error in the condition stmt of a loop: %v", err)
	}
	scope.expressions = append(scope.expressions, ifStmt)

	err = scope.parseStatementList(stmt.Body.List, indent+2)
	if err != nil {
		return nil, fmt.Errorf("error in the body of a loop: %v", err)
	}

	if stmt.Post != nil {
		post := []ast.Stmt{stmt.Post}
		err = scope.parseStatementList(post, indent+2)
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
	scope := s.f.createScope("block")
	err := scope.parseStatementList(stmt.List, indent+1)
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
	cond, err := s.parseExpr(stmt.Cond, nil, indent+1)
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
		vRHS, err := s.parseIdent(x, indent+2)
		if err != nil {
			return nil, fmt.Errorf("error in IncDecStmt: %v", err)
		}

		inc, err := s.createLiteral("1", v.getType(), indent+2)
		if err != nil {
			return nil, fmt.Errorf("error in IncDecStmt: %v", err)
		}

		rhs, err := s.createBinaryExpr(vRHS, inc, binOpMapping[stmt.Tok], v.getType(), indent+1)
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
		value, err := s.parseExpr(stmt.Results[0], s.f.result.t, indent+1)
		if err != nil {
			return nil, err
		}
		r.value = value
	}
	return r, nil
}

func (n *WasmNop) getType() WasmTypeI {
	return nil
}

func (n *WasmNop) print(writer FormattingWriter) {
	writer.PrintfIndent(n.getIndent(), "(nop)\n")
}

func (n *WasmNop) getNode() ast.Node {
	return nil
}

func (b *WasmBlock) getType() WasmTypeI {
	return nil
}

func (b *WasmBlock) print(writer FormattingWriter) {
	writer.PrintfIndent(b.getIndent(), "(block\n")
	for _, expr := range b.scope.expressions {
		expr.print(writer)
	}
	writer.PrintfIndent(b.getIndent(), ") ;; block\n")
}

func (b *WasmBlock) getNode() ast.Node {
	if b.stmt == nil {
		return nil
	} else {
		return b.stmt
	}
}

func (r *WasmReturn) getType() WasmTypeI {
	if r.value == nil {
		return nil
	} else {
		return r.value.getType()
	}
}

func (r *WasmReturn) print(writer FormattingWriter) {
	writer.PrintfIndent(r.getIndent(), "(return\n")
	if r.value != nil {
		r.value.print(writer)
	}
	writer.PrintfIndent(r.getIndent(), ") ;; return\n")
}

func (r *WasmReturn) getNode() ast.Node {
	if r.stmt == nil {
		return nil
	} else {
		return r.stmt
	}
}

func (i *WasmIf) getType() WasmTypeI {
	return nil
}

func (i *WasmIf) print(writer FormattingWriter) {
	writer.PrintfIndent(i.getIndent(), "(if\n")
	i.cond.print(writer)
	i.body.print(writer)
	if i.bodyElse != nil {
		i.bodyElse.print(writer)
	}
	writer.PrintfIndent(i.getIndent(), ") ;; if\n")
}

func (i *WasmIf) getNode() ast.Node {
	if i.stmt == nil {
		return nil
	} else {
		return i.stmt
	}
}

func (l *WasmLoop) getType() WasmTypeI {
	return nil
}

func (l *WasmLoop) print(writer FormattingWriter) {
	writer.PrintfIndent(l.getIndent(), "(loop $%s\n", l.scope.name)
	for _, e := range l.scope.expressions {
		e.print(writer)
	}
	writer.PrintfIndent(l.getIndent(), ") ;; loop\n")
}

func (l *WasmLoop) getNode() ast.Node {
	if l.stmt == nil {
		return nil
	} else {
		return l.stmt
	}
}

func (b *WasmBreak) getType() WasmTypeI {
	return nil
}

func (b *WasmBreak) print(writer FormattingWriter) {
	writer.PrintfIndent(b.getIndent(), "(break $%s)\n", b.scope.name)
}

func (b *WasmBreak) getNode() ast.Node {
	return nil
}

func (s *WasmSetLocal) print(writer FormattingWriter) {
	writer.PrintfIndent(s.getIndent(), "(set_local %s\n", s.lhs.getName())
	s.rhs.print(writer)
	writer.PrintfIndent(s.getIndent(), ") ;; set_local %s\n", s.lhs.getName())
}

func (s *WasmSetLocal) getType() WasmTypeI {
	return s.lhs.getType()
}

func (s *WasmSetLocal) getNode() ast.Node {
	if s.stmt == nil {
		return nil
	} else {
		return s.stmt
	}
}
