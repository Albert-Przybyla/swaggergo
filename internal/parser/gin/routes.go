package gin

import "go/ast"

type Route struct {
	Method  string
	Path    string
	Group   string
	Handler string
}

func parseRoutes(file *ast.File, groups map[string]string) []Route {
	var routes []Route

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		method := sel.Sel.Name
		if !isHTTPMethod(method) {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		groupName := ident.Name
		groupPath, ok := groups[groupName]
		if !ok {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		pathLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}

		handler := ""
		if len(call.Args) > 1 {
			if sel, ok := call.Args[len(call.Args)-1].(*ast.SelectorExpr); ok {
				handler = sel.Sel.Name
			}
		}

		routes = append(routes, Route{
			Method:  method,
			Path:    trimQuotes(pathLit.Value),
			Group:   groupPath,
			Handler: handler,
		})

		return true
	})

	return routes
}
