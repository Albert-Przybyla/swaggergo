package gin

func findHandlerDecl(ctx *packageContext, name string) *functionInfo {
	candidates := ctx.funcs[name]
	if len(candidates) == 0 {
		return nil
	}

	return candidates[0]
}
