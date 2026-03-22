package gin

import (
	"go/ast"
	"strconv"
	"strings"
)

type RouteResponse struct {
	StatusCode string
	TypeName   string
	Schema     *Schema
}

func extractResponse(fn *functionInfo, ctx *packageContext) RouteResponse {
	if fn == nil || fn.decl == nil || fn.decl.Body == nil {
		return RouteResponse{}
	}

	extractor := &responseExtractor{
		fn:       fn,
		ctx:      ctx,
		resolver: newFunctionResolverWithContext(fn, ctx),
		visited:  make(map[string]bool),
	}

	ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		response, ok := extractor.responseFromJSONCall(call)
		if !ok {
			return true
		}

		extractor.response = response
		return false
	})

	return extractor.response
}

type responseExtractor struct {
	fn       *functionInfo
	ctx      *packageContext
	resolver *functionResolver
	visited  map[string]bool
	response RouteResponse
}

func (e *responseExtractor) responseFromJSONCall(call *ast.CallExpr) (RouteResponse, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "JSON" || len(call.Args) < 2 {
		return RouteResponse{}, false
	}

	statusCode := resolveStatusCode(call.Args[0])
	typeName := trimPointer(e.resolveResponseType(call.Args[1], e.fn, e.resolver))
	inlineSchema := e.inlineSchemaForExpr(call.Args[1], e.resolver)
	if typeName == "" || isBuiltinTypeName(typeName) {
		return RouteResponse{StatusCode: statusCode, Schema: inlineSchema}, true
	}

	return RouteResponse{
		StatusCode: statusCode,
		TypeName:   typeName,
		Schema:     inlineSchema,
	}, true
}

func (e *responseExtractor) resolveResponseType(expr ast.Expr, currentFn *functionInfo, resolver *functionResolver) string {
	switch typed := expr.(type) {
	case *ast.UnaryExpr:
		if typed.Op.String() == "&" {
			return e.resolveResponseType(typed.X, currentFn, resolver)
		}
	case *ast.CompositeLit:
		return trimPointer(exprTypeNameForFile(typed.Type, currentFn.file))
	case *ast.Ident:
		if resolver != nil {
			if src := resolver.sourceExpr(typed.Name); src != nil && src != typed {
				return e.resolveResponseType(src, currentFn, resolver)
			}

			if typeName := trimPointer(resolver.resolveExprType(typed)); typeName != "" {
				return typeName
			}
		}
	case *ast.CallExpr:
		fn := resolveCalledFunction(e.ctx, currentFn, typed, resolver)
		if fn == nil {
			if resolver != nil {
				return trimPointer(resolver.resolveExprType(typed))
			}
			return ""
		}
		return trimPointer(e.resolveFunctionResultType(fn, 0))
	}

	if resolver != nil {
		return trimPointer(resolver.resolveExprType(expr))
	}

	return ""
}

func (e *responseExtractor) resolveFunctionResultType(fn *functionInfo, resultIndex int) string {
	key := functionKey(fn, resultIndex)
	if e.visited[key] {
		return resultTypeAtIndex(fn, resultIndex)
	}
	e.visited[key] = true
	defer delete(e.visited, key)

	if declared := resultTypeAtIndex(fn, resultIndex); declared != "" {
		return declared
	}

	if inferred := inferFunctionResultType(fn, e.ctx, resultIndex, e.visited); inferred != "" {
		return inferred
	}

	return resultTypeAtIndex(fn, resultIndex)
}

func inferFunctionResultType(fn *functionInfo, ctx *packageContext, resultIndex int, visited map[string]bool) string {
	if fn == nil || fn.decl == nil || fn.decl.Body == nil {
		return ""
	}

	resolver := newFunctionResolverWithContext(fn, ctx)
	var inferred string

	ast.Inspect(fn.decl.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || resultIndex >= len(ret.Results) {
			return true
		}

		typeName := inferReturnExprType(ret.Results[resultIndex], fn, ctx, resolver, resultIndex, visited)
		if typeName == "" {
			return true
		}

		if inferred == "" {
			inferred = typeName
		}

		return inferred == ""
	})

	return inferred
}

func inferReturnExprType(expr ast.Expr, fn *functionInfo, ctx *packageContext, resolver *functionResolver, resultIndex int, visited map[string]bool) string {
	switch typed := expr.(type) {
	case *ast.CompositeLit:
		return trimPointer(exprTypeNameForFile(typed.Type, fn.file))
	case *ast.UnaryExpr:
		if typed.Op.String() == "&" {
			return inferReturnExprType(typed.X, fn, ctx, resolver, resultIndex, visited)
		}
	case *ast.Ident:
		if typed.Name == "nil" {
			return ""
		}
		if src := resolver.sourceExpr(typed.Name); src != nil && src != typed {
			return inferReturnExprType(src, fn, ctx, resolver, resultIndex, visited)
		}
		return trimPointer(resolver.resolveExprType(typed))
	case *ast.CallExpr:
		callee := resolveCalledFunction(ctx, fn, typed, resolver)
		if callee == nil {
			return trimPointer(resolver.resolveExprType(typed))
		}

		key := functionKey(callee, resultIndex)
		if visited[key] {
			return resultTypeAtIndex(callee, resultIndex)
		}
		visited[key] = true
		defer delete(visited, key)

		if declared := resultTypeAtIndex(callee, resultIndex); declared != "" {
			return declared
		}

		if inferred := inferFunctionResultType(callee, ctx, resultIndex, visited); inferred != "" {
			return inferred
		}
		return resultTypeAtIndex(callee, resultIndex)
	}

	return trimPointer(resolver.resolveExprType(expr))
}

func resolveStatusCode(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		if typed.Kind.String() == "INT" {
			return typed.Value
		}
	case *ast.SelectorExpr:
		if code, ok := httpStatusCodes[typed.Sel.Name]; ok {
			return code
		}
	}

	return "200"
}

func (e *responseExtractor) inlineSchemaForExpr(expr ast.Expr, resolver *functionResolver) *Schema {
	switch typed := expr.(type) {
	case *ast.UnaryExpr:
		if typed.Op.String() == "&" {
			return e.inlineSchemaForExpr(typed.X, resolver)
		}
	case *ast.Ident:
		if resolver != nil {
			if src := resolver.sourceExpr(typed.Name); src != nil && src != typed {
				return e.inlineSchemaForExpr(src, resolver)
			}
		}
	case *ast.CompositeLit:
		typeName := trimPointer(exprTypeNameForFile(typed.Type, e.fn.file))
		if e.hasSchemaForType(typeName) {
			return nil
		}
		return schemaFromCompositeLiteral(typed, e.fn.file, e.ctx)
	}

	return nil
}

func (e *responseExtractor) hasSchemaForType(typeName string) bool {
	typeName = normalizeSchemaTypeName(typeName)
	if typeName == "" || e.ctx == nil {
		return false
	}
	if _, ok := e.ctx.structs[typeName]; ok {
		return true
	}
	baseName, _ := splitGenericTypeName(typeName)
	_, ok := e.ctx.structs[baseName]
	return ok
}

var httpStatusCodes = map[string]string{
	"StatusOK":                  "200",
	"StatusCreated":             "201",
	"StatusAccepted":            "202",
	"StatusNoContent":           "204",
	"StatusBadRequest":          "400",
	"StatusUnauthorized":        "401",
	"StatusForbidden":           "403",
	"StatusNotFound":            "404",
	"StatusConflict":            "409",
	"StatusUnprocessableEntity": "422",
	"StatusInternalServerError": "500",
}

func flattenResultTypes(fields []*ast.Field, currentFile *fileInfo) []string {
	var types []string
	for _, field := range fields {
		typeName := exprTypeNameForFile(field.Type, currentFile)
		if len(field.Names) == 0 {
			types = append(types, typeName)
			continue
		}
		for range field.Names {
			types = append(types, typeName)
		}
	}
	return types
}

func resultTypeAtIndex(fn *functionInfo, resultIndex int) string {
	if fn == nil || fn.decl == nil || fn.decl.Type == nil || fn.decl.Type.Results == nil {
		return ""
	}

	flat := flattenResultTypes(fn.decl.Type.Results.List, fn.file)
	if resultIndex < 0 || resultIndex >= len(flat) {
		return ""
	}

	return flat[resultIndex]
}

func resolveCalledFunction(ctx *packageContext, currentFn *functionInfo, call *ast.CallExpr, resolver *functionResolver) *functionInfo {
	if ctx == nil || currentFn == nil || currentFn.file == nil {
		return nil
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return findFunctionInPackage(ctx, currentFn.file.pkg, fun.Name)
	case *ast.SelectorExpr:
		if pkgIdent, ok := fun.X.(*ast.Ident); ok {
			if importPath, exists := currentFn.file.imports[pkgIdent.Name]; exists {
				return findFunctionByImportPath(ctx, importPath, fun.Sel.Name)
			}
		}

		ref := extractHandlerReference(fun, resolver)
		if ref.Name == "" {
			return nil
		}
		return findMethodInContext(ctx, ref)
	}

	return nil
}

func findFunctionInPackage(ctx *packageContext, pkg *packageInfo, name string) *functionInfo {
	if ctx == nil {
		return nil
	}

	for _, candidate := range ctx.funcs[name] {
		if receiverTypeName(candidate) != "" {
			continue
		}
		if samePackage(candidate.file, pkg) {
			return candidate
		}
	}

	for _, candidate := range ctx.funcs[name] {
		if receiverTypeName(candidate) == "" {
			return candidate
		}
	}

	return nil
}

func findFunctionByImportPath(ctx *packageContext, importPath string, name string) *functionInfo {
	if ctx == nil {
		return nil
	}

	for _, candidate := range ctx.funcs[name] {
		if receiverTypeName(candidate) != "" || candidate.file == nil || candidate.file.pkg == nil {
			continue
		}
		if candidate.file.pkg.importPath == importPath {
			return candidate
		}
	}

	return nil
}

func findMethodInContext(ctx *packageContext, ref handlerReference) *functionInfo {
	if ctx == nil {
		return nil
	}

	return findHandlerDecl(ctx, ref)
}

func samePackage(file *fileInfo, pkg *packageInfo) bool {
	if file == nil || file.pkg == nil || pkg == nil {
		return false
	}

	if file.pkg.importPath != "" && pkg.importPath != "" {
		return file.pkg.importPath == pkg.importPath
	}

	return file.pkg.name == pkg.name && file.pkg.dir == pkg.dir
}

func exprTypeNameForFile(expr ast.Expr, currentFile *fileInfo) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		name := exprTypeNameForFile(typed.X, currentFile)
		if name == "" {
			return ""
		}
		return "*" + name
	case *ast.SelectorExpr:
		left, ok := typed.X.(*ast.Ident)
		if ok && currentFile != nil {
			if importPath, exists := currentFile.imports[left.Name]; exists {
				if currentFile.pkg != nil && importPath == currentFile.pkg.importPath {
					return typed.Sel.Name
				}
			}
		}
		return selectorName(typed)
	case *ast.ArrayType:
		name := exprTypeNameForFile(typed.Elt, currentFile)
		if name == "" {
			return ""
		}
		return "[]" + name
	case *ast.IndexExpr:
		base := exprTypeNameForFile(typed.X, currentFile)
		arg := exprTypeNameForFile(typed.Index, currentFile)
		if base == "" || arg == "" {
			return ""
		}
		return base + "[" + arg + "]"
	case *ast.IndexListExpr:
		base := exprTypeNameForFile(typed.X, currentFile)
		if base == "" {
			return ""
		}
		args := make([]string, 0, len(typed.Indices))
		for _, idx := range typed.Indices {
			arg := exprTypeNameForFile(idx, currentFile)
			if arg == "" {
				return ""
			}
			args = append(args, arg)
		}
		return base + "[" + strings.Join(args, ",") + "]"
	case *ast.MapType:
		return "map"
	case *ast.CompositeLit:
		return exprTypeNameForFile(typed.Type, currentFile)
	default:
		return ""
	}
}

func schemaFromCompositeLiteral(lit *ast.CompositeLit, currentFile *fileInfo, ctx *packageContext) *Schema {
	if lit == nil {
		return nil
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		name := compositeLiteralFieldName(kv.Key)
		if name == "" {
			continue
		}

		fieldSchema := schemaFromValueExpr(kv.Value, currentFile, ctx)
		if fieldSchema == nil {
			continue
		}

		schema.Properties[name] = fieldSchema
		schema.Required = append(schema.Required, name)
	}

	if len(schema.Properties) == 0 {
		return nil
	}

	return schema
}

func compositeLiteralFieldName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

func schemaFromValueExpr(expr ast.Expr, currentFile *fileInfo, ctx *packageContext) *Schema {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		switch typed.Kind.String() {
		case "STRING":
			return &Schema{Type: "string"}
		case "INT":
			return &Schema{Type: "integer", Format: "int32"}
		case "FLOAT":
			return &Schema{Type: "number", Format: "double"}
		default:
			return &Schema{Type: "object"}
		}
	case *ast.Ident:
		switch typed.Name {
		case "true", "false":
			return &Schema{Type: "boolean"}
		case "nil":
			return &Schema{Type: "object"}
		default:
			if schema, ok := builtinSchema(typed.Name); ok {
				return schema
			}
			return &Schema{Type: "object"}
		}
	case *ast.UnaryExpr:
		if typed.Op.String() == "&" {
			return schemaFromValueExpr(typed.X, currentFile, ctx)
		}
	case *ast.CompositeLit:
		if typed.Type != nil {
			if schema := schemaFromExpr(typed.Type, currentFile, nil, map[string]bool{}, ctx); schema != nil && schema.Ref != "" {
				return schema
			}
		}
		return schemaFromCompositeLiteral(typed, currentFile, ctx)
	case *ast.ArrayType:
		return &Schema{Type: "array"}
	case *ast.CallExpr:
		return &Schema{Type: "object"}
	}

	if schema := schemaFromExpr(expr, currentFile, nil, map[string]bool{}, ctx); schema != nil {
		return schema
	}

	return &Schema{Type: "object"}
}

func functionKey(fn *functionInfo, resultIndex int) string {
	name := ""
	if fn != nil && fn.decl != nil {
		name = fn.decl.Name.Name
	}
	if recv := receiverTypeName(fn); recv != "" {
		name = recv + "." + name
	}
	if fn != nil && fn.file != nil && fn.file.path != "" {
		name = fn.file.path + ":" + name
	}
	return name + ":" + strconv.Itoa(resultIndex)
}
