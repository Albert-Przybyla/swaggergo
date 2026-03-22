package gin

import (
	"go/ast"
	"strings"
)

type functionResolver struct {
	fn       *functionInfo
	ctx      *packageContext
	varTypes map[string]string
	varExprs map[string]ast.Expr
	callRefs map[*ast.CallExpr]*callReference
}

type callReference struct {
	call        *ast.CallExpr
	resultIndex int
}

func newFunctionResolver(fn *functionInfo) *functionResolver {
	return newFunctionResolverWithContext(fn, nil)
}

func newFunctionResolverWithContext(fn *functionInfo, ctx *packageContext) *functionResolver {
	resolver := &functionResolver{
		fn:       fn,
		ctx:      ctx,
		varTypes: make(map[string]string),
		varExprs: make(map[string]ast.Expr),
		callRefs: make(map[*ast.CallExpr]*callReference),
	}

	resolver.indexFuncParams()

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

func (r *functionResolver) indexFuncParams() {
	if r.fn == nil || r.fn.decl == nil {
		return
	}

	if r.fn.decl.Recv != nil {
		for _, field := range r.fn.decl.Recv.List {
			typeName := r.exprTypeName(field.Type)
			for _, name := range field.Names {
				r.varTypes[name.Name] = typeName
				r.varExprs[name.Name] = name
			}
		}
	}

	if r.fn.decl.Type == nil || r.fn.decl.Type.Params == nil {
		return
	}

	for _, field := range r.fn.decl.Type.Params.List {
		typeName := r.exprTypeName(field.Type)
		for _, name := range field.Names {
			r.varTypes[name.Name] = typeName
			r.varExprs[name.Name] = name
		}
	}
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
				r.varExprs[name.Name] = name
				continue
			}

			if i < len(valueSpec.Values) {
				r.varExprs[name.Name] = valueSpec.Values[i]
				if typeName := r.resolveExprType(valueSpec.Values[i]); typeName != "" {
					r.varTypes[name.Name] = typeName
				}
				if call, ok := valueSpec.Values[i].(*ast.CallExpr); ok {
					r.callRefs[call] = &callReference{call: call, resultIndex: i}
				}
			}
		}
	}
}

func (r *functionResolver) indexAssignStmt(stmt *ast.AssignStmt) {
	if len(stmt.Rhs) == 1 {
		if call, ok := stmt.Rhs[0].(*ast.CallExpr); ok {
			for i, lhs := range stmt.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name == "_" {
					continue
				}

				r.varExprs[ident.Name] = call
				r.callRefs[call] = &callReference{call: call, resultIndex: i}
				if typeName := r.resolveCallResultType(call, i); typeName != "" {
					r.varTypes[ident.Name] = typeName
					continue
				}
				if typeName := r.resolveExprType(call); typeName != "" {
					r.varTypes[ident.Name] = typeName
				}
			}
			return
		}
	}

	for i, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" || i >= len(stmt.Rhs) {
			continue
		}

		r.varExprs[ident.Name] = stmt.Rhs[i]
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
		if ident, ok := typed.Fun.(*ast.Ident); ok {
			switch ident.Name {
			case "new":
				if len(typed.Args) == 1 {
					return r.exprTypeName(typed.Args[0])
				}
			case "make":
				if len(typed.Args) > 0 {
					return r.exprTypeName(typed.Args[0])
				}
			case "append":
				if len(typed.Args) > 0 {
					return r.resolveExprType(typed.Args[0])
				}
			}
		}
		if ref := r.callRefs[typed]; ref != nil {
			return r.resolveCallResultType(typed, ref.resultIndex)
		}
		return r.resolveCallResultType(typed, 0)
	case *ast.SelectorExpr:
		if typeName := r.resolveSelectorType(typed); typeName != "" {
			return typeName
		}
	}

	return r.exprTypeName(expr)
}

func (r *functionResolver) resolveSelectorType(sel *ast.SelectorExpr) string {
	if r == nil || r.ctx == nil {
		return ""
	}

	baseType := trimPointer(r.resolveExprType(sel.X))
	if baseType == "" {
		return ""
	}

	info := r.ctx.structs[baseType]
	if info == nil && r.fn != nil && r.fn.file != nil && r.fn.file.pkg != nil {
		info = r.ctx.structs[r.fn.file.pkg.name+"."+baseType]
	}
	if info == nil {
		return ""
	}

	for _, field := range info.structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == sel.Sel.Name {
				return r.exprTypeName(field.Type)
			}
		}
	}

	return ""
}

func (r *functionResolver) resolveCallResultType(call *ast.CallExpr, resultIndex int) string {
	if r == nil || call == nil {
		return ""
	}
	return ""
}

func (r *functionResolver) sourceExpr(name string) ast.Expr {
	if r == nil {
		return nil
	}
	expr := r.varExprs[name]
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
		return nil
	}
	return expr
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
	case *ast.ArrayType:
		name := r.exprTypeName(typed.Elt)
		if name == "" {
			return ""
		}
		return "[]" + name
	case *ast.IndexExpr:
		base := r.exprTypeName(typed.X)
		arg := r.exprTypeName(typed.Index)
		if base == "" || arg == "" {
			return ""
		}
		return base + "[" + arg + "]"
	case *ast.IndexListExpr:
		base := r.exprTypeName(typed.X)
		if base == "" {
			return ""
		}
		args := make([]string, 0, len(typed.Indices))
		for _, idx := range typed.Indices {
			arg := r.exprTypeName(idx)
			if arg == "" {
				return ""
			}
			args = append(args, arg)
		}
		return base + "[" + strings.Join(args, ",") + "]"
	case *ast.MapType:
		return "map"
	case *ast.CompositeLit:
		return r.exprTypeName(typed.Type)
	default:
		return ""
	}
}
