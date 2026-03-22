package gin

import "go/ast"

type Route struct {
	Method  string
	Path    string
	Group   string
	Handler string

	Summary     string
	Description string
	QueryParams []QueryParam
	BodyType    string
	Response    RouteResponse
}

func parseRoutes(ctx *packageContext, groups map[string]string) []Route {
	var routes []Route

	for _, decl := range ctx.routerFile.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Body == nil {
			continue
		}

		resolver := newFunctionResolver(&functionInfo{
			decl: fnDecl,
			file: ctx.routerFileInfo,
		})

		ast.Inspect(fnDecl.Body, func(n ast.Node) bool {
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

			handlerRef := handlerReference{}
			if len(call.Args) > 1 {
				handlerRef = extractHandlerReference(call.Args[len(call.Args)-1], resolver)
			}

			fn := findHandlerDecl(ctx, handlerRef)

			var queryParams []QueryParam
			var bodyType string
			var summary string
			var description string
			var response RouteResponse

			if fn != nil {
				queryParams = extractQueryParams(fn, ctx)
				bodyType = extractBodyType(fn)
				summary, description = parseCommentMetadata(fn.decl.Doc)
				response = extractResponse(fn, ctx)
			}

			routes = append(routes, Route{
				Method:      method,
				Path:        trimQuotes(pathLit.Value),
				Group:       groupPath,
				Handler:     handlerRef.Name,
				Summary:     summary,
				Description: description,
				QueryParams: queryParams,
				BodyType:    bodyType,
				Response:    response,
			})

			return true
		})
	}

	return routes
}

func extractHandlerName(expr ast.Expr) string {
	return extractHandlerReference(expr, nil).Name
}

func extractHandlerReference(expr ast.Expr, resolver *functionResolver) handlerReference {
	switch typed := expr.(type) {
	case *ast.Ident:
		return handlerReference{Name: typed.Name}
	case *ast.SelectorExpr:
		ref := handlerReference{Name: typed.Sel.Name}
		if resolver != nil {
			ref.ReceiverType = trimPointer(resolver.resolveExprType(typed.X))
		}
		return ref
	default:
		return handlerReference{}
	}
}
