package gin

import "go/ast"

func extractBodyType(fn *functionInfo) string {
	if fn == nil || fn.decl == nil || fn.decl.Body == nil {
		return ""
	}

	resolver := newFunctionResolver(fn)
	var bodyType string

	ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if len(call.Args) > 0 {
			if isBodyBindMethod(sel.Sel.Name) {
				bodyType = trimPointer(resolver.resolveExprType(call.Args[0]))
				return false
			}
			if len(call.Args) > 1 {
				if isHttpxBodyBindMethod(sel.Sel.Name) {
					bodyType = trimPointer(resolver.resolveExprType(call.Args[1]))
					return false
				}
			}
		}
		return true
	})

	return bodyType
}

func isBodyBindMethod(name string) bool {
	switch name {
	case "Bind", "BindJSON", "MustBind", "MustBindWith", "ShouldBind", "ShouldBindBodyWith", "ShouldBindJSON", "ShouldBindWith":
		return true
	default:
		return false
	}
}

func isHttpxBodyBindMethod(name string) bool {
	switch name {
	case "ValidateStruct":
		return true
	default:
		return false
	}
}
