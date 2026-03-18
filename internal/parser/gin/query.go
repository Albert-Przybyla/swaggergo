package gin

type QueryParam struct {
	Name     string
	Type     string
	Format   string
	Required bool
	TypeName string
}

func extractQueryParams(fn *functionInfo, ctx *packageContext) []QueryParam {
	if fn == nil || fn.decl == nil || fn.decl.Body == nil {
		return nil
	}

	extractor := newQueryExtractor(fn, ctx)
	return extractor.extract()
}
