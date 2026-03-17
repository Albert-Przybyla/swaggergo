package gin

import "go/ast"

func findHandlerDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if fn.Name.Name == name {
			return fn
		}
	}
	return nil
}
