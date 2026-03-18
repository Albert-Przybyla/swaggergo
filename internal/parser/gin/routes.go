package gin

import "go/ast"

type Route struct {
	Method  string
	Path    string
	Group   string
	Handler string

	QueryParams []QueryParam
	BodyType    string
}

func parseRoutes(ctx *packageContext, groups map[string]string) []Route {
	var routes []Route

	ast.Inspect(ctx.routerFile, func(n ast.Node) bool {
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
			handler = extractHandlerName(call.Args[len(call.Args)-1])
		}

		fn := findHandlerDecl(ctx, handler)

		var queryParams []QueryParam
		var bodyType string

		if fn != nil {
			queryParams = extractQueryParams(fn, ctx)
			bodyType = extractBodyType(fn)
		}

		routes = append(routes, Route{
			Method:      method,
			Path:        trimQuotes(pathLit.Value),
			Group:       groupPath,
			Handler:     handler,
			QueryParams: queryParams,
			BodyType:    bodyType,
		})

		return true
	})

	return routes
}

func extractHandlerName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}
