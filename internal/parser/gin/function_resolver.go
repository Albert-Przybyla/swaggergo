package gin

import "go/ast"

type functionResolver struct {
	fn       *functionInfo
	varTypes map[string]string
}

func newFunctionResolver(fn *functionInfo) *functionResolver {
	resolver := &functionResolver{
		fn:       fn,
		varTypes: make(map[string]string),
	}

	if fn == nil || fn.decl == nil || fn.decl.Body == nil {
		return resolver
	}

	ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
		switch typed := n.(type) {
		case *ast.DeclStmt:
			resolver.indexDeclStmt(typed)
		case *ast.AssignStmt:
			resolver.indexAssignStmt(typed)
		}

		return true
	})

	return resolver
}

func (r *functionResolver) indexDeclStmt(stmt *ast.DeclStmt) {
	gen, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || gen.Tok.String() != "var" {
		return
	}

	for _, spec := range gen.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range valueSpec.Names {
			if valueSpec.Type != nil {
				r.varTypes[name.Name] = r.exprTypeName(valueSpec.Type)
				continue
			}

			if i < len(valueSpec.Values) {
				if typeName := r.resolveExprType(valueSpec.Values[i]); typeName != "" {
					r.varTypes[name.Name] = typeName
				}
			}
		}
	}
}

func (r *functionResolver) indexAssignStmt(stmt *ast.AssignStmt) {
	for i, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" || i >= len(stmt.Rhs) {
			continue
		}

		if typeName := r.resolveExprType(stmt.Rhs[i]); typeName != "" {
			r.varTypes[ident.Name] = typeName
		}
	}
}

func (r *functionResolver) resolveExprType(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.UnaryExpr:
		if typed.Op.String() == "&" {
			return r.resolveExprType(typed.X)
		}
	case *ast.Ident:
		if typeName, ok := r.varTypes[typed.Name]; ok {
			return typeName
		}
		return typed.Name
	case *ast.CompositeLit:
		return r.exprTypeName(typed.Type)
	case *ast.CallExpr:
		if ident, ok := typed.Fun.(*ast.Ident); ok && ident.Name == "new" && len(typed.Args) == 1 {
			return r.exprTypeName(typed.Args[0])
		}
	}

	return r.exprTypeName(expr)
}

func (r *functionResolver) exprTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		name := r.exprTypeName(typed.X)
		if name == "" {
			return ""
		}
		return "*" + name
	case *ast.SelectorExpr:
		left, ok := typed.X.(*ast.Ident)
		if ok && r.fn != nil && r.fn.file != nil {
			if importPath, exists := r.fn.file.imports[left.Name]; exists {
				if pkg := r.fn.file.pkg; pkg != nil && importPath == pkg.importPath {
					return typed.Sel.Name
				}
			}
		}
		return selectorName(typed)
	case *ast.CompositeLit:
		return r.exprTypeName(typed.Type)
	default:
		return ""
	}
}
