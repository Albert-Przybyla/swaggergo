package gin

import "go/ast"

type handlerReference struct {
	Name         string
	ReceiverType string
}

func findHandlerDecl(ctx *packageContext, ref handlerReference) *functionInfo {
	candidates := ctx.funcs[ref.Name]
	if len(candidates) == 0 {
		return nil
	}

	if ref.ReceiverType != "" {
		want := shortTypeName(ref.ReceiverType)
		for _, candidate := range candidates {
			if shortTypeName(receiverTypeName(candidate)) == want {
				return candidate
			}
		}
	}

	return candidates[0]
}

func receiverTypeName(fn *functionInfo) string {
	if fn == nil || fn.decl == nil || fn.decl.Recv == nil || len(fn.decl.Recv.List) == 0 {
		return ""
	}

	return trimPointer(handlerExprTypeName(fn.decl.Recv.List[0].Type))
}

func handlerExprTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return handlerExprTypeName(typed.X)
	case *ast.SelectorExpr:
		return selectorName(typed)
	default:
		return ""
	}
}

func shortTypeName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return name
}
