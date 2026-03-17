package gin

import "go/ast"

func extractQueryParams(fn *ast.FuncDecl) []string {
	if fn == nil || fn.Body == nil {
		return nil
	}
	var params []string

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name != "Query" && sel.Sel.Name != "DefaultQuery" {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		arg, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}

		params = append(params, trimQuotes(arg.Value))
		return true
	})

	return params
}
