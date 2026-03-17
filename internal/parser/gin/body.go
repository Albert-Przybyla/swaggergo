package gin

import "go/ast"

func extractBodyType(fn *ast.FuncDecl) string {
	if fn == nil || fn.Body == nil {
		return ""
	}
	var bodyType string

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name != "BindJSON" && sel.Sel.Name != "ShouldBindJSON" {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		unary, ok := call.Args[0].(*ast.UnaryExpr)
		if !ok {
			return true
		}

		ident, ok := unary.X.(*ast.Ident)
		if !ok {
			return true
		}

		bodyType = ident.Name
		return false
	})

	return bodyType
}
