package gin

import "go/ast"

func parseGroups(file *ast.File) map[string]string {
	groups := map[string]string{}

	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}

		ident, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}

		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Group" {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		arg, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}

		parentPath := ""
		if baseIdent, ok := sel.X.(*ast.Ident); ok {
			parentPath = groups[baseIdent.Name]
		}

		groups[ident.Name] = join(parentPath, trimQuotes(arg.Value))
		return true
	})

	return groups
}
