package gin

import (
	"go/ast"
	"go/token"
)

type queryExtractor struct {
	fn            *functionInfo
	ctx           *packageContext
	resolver      *functionResolver
	paramsByName  map[string]*QueryParam
	varToParam    map[string]string
	orderedParams []string
}

func newQueryExtractor(fn *functionInfo, ctx *packageContext) *queryExtractor {
	return &queryExtractor{
		fn:           fn,
		ctx:          ctx,
		resolver:     newFunctionResolver(fn),
		paramsByName: make(map[string]*QueryParam),
		varToParam:   make(map[string]string),
	}
}

func (q *queryExtractor) extract() []QueryParam {
	ast.Inspect(q.fn.decl.Body, func(n ast.Node) bool {
		switch typed := n.(type) {
		case *ast.AssignStmt:
			q.inspectAssignStmt(typed)
		case *ast.CallExpr:
			q.inspectCallExpr(typed)
		}

		return true
	})

	params := make([]QueryParam, 0, len(q.orderedParams))
	for _, name := range q.orderedParams {
		params = append(params, *q.paramsByName[name])
	}

	return params
}

func (q *queryExtractor) inspectAssignStmt(stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}

		if paramName, ok := queryCallParamName(call); ok {
			param := q.ensureParam(paramName)
			if schema, ok := schemaForQueryMethod(call); ok {
				param.Type = schema.Type
				param.Format = schema.Format
			}

			if i < len(stmt.Lhs) {
				if ident, ok := stmt.Lhs[i].(*ast.Ident); ok && ident.Name != "_" {
					q.varToParam[ident.Name] = paramName
				}
			}
			continue
		}

		if paramName, ok := q.paramNameFromConversion(call); ok {
			param := q.ensureParam(paramName)
			if schema, ok := schemaForConversion(call); ok {
				param.Type = schema.Type
				param.Format = schema.Format
			}
		}
	}
}

func (q *queryExtractor) inspectCallExpr(call *ast.CallExpr) {
	if paramName, ok := queryCallParamName(call); ok {
		param := q.ensureParam(paramName)
		if schema, ok := schemaForQueryMethod(call); ok {
			param.Type = schema.Type
			param.Format = schema.Format
		}
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !isQueryBindMethod(sel.Sel.Name) || len(call.Args) == 0 {
		return
	}

	typeName := trimPointer(q.resolver.resolveExprType(call.Args[0]))
	if typeName == "" {
		return
	}

	for _, param := range buildQueryParamsFromType(typeName, q.fn.file, q.ctx) {
		existing := q.ensureParam(param.Name)
		existing.Type = param.Type
		existing.Format = param.Format
		existing.Required = param.Required
		existing.TypeName = param.TypeName
	}
}

func (q *queryExtractor) ensureParam(name string) *QueryParam {
	if existing, ok := q.paramsByName[name]; ok {
		return existing
	}

	param := &QueryParam{
		Name: name,
		Type: "string",
	}
	q.paramsByName[name] = param
	q.orderedParams = append(q.orderedParams, name)
	return param
}

func (q *queryExtractor) paramNameFromConversion(call *ast.CallExpr) (string, bool) {
	for _, arg := range call.Args {
		switch typed := arg.(type) {
		case *ast.CallExpr:
			if name, ok := queryCallParamName(typed); ok {
				return name, true
			}
		case *ast.Ident:
			if name, ok := q.varToParam[typed.Name]; ok {
				return name, true
			}
		}
	}

	return "", false
}

func queryCallParamName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) == 0 {
		return "", false
	}

	if !isQueryMethod(sel.Sel.Name) {
		return "", false
	}

	arg, ok := call.Args[0].(*ast.BasicLit)
	if !ok || arg.Kind != token.STRING {
		return "", false
	}

	return trimQuotes(arg.Value), true
}

func isQueryMethod(name string) bool {
	switch name {
	case "DefaultQuery", "GetQuery", "GetQueryArray", "GetQueryMap", "Query", "QueryArray", "QueryMap":
		return true
	default:
		return false
	}
}

func isQueryBindMethod(name string) bool {
	switch name {
	case "BindQuery", "ShouldBindQuery":
		return true
	default:
		return false
	}
}

func schemaForQueryMethod(call *ast.CallExpr) (*Schema, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	switch sel.Sel.Name {
	case "QueryArray", "GetQueryArray":
		return &Schema{Type: "array", Items: &Schema{Type: "string"}}, true
	case "QueryMap", "GetQueryMap":
		return &Schema{Type: "object"}, true
	default:
		return &Schema{Type: "string"}, true
	}
}

func schemaForConversion(call *ast.CallExpr) (*Schema, bool) {
	name := calledFunctionName(call.Fun)
	switch name {
	case "strconv.Atoi", "strconv.ParseInt", "strconv.ParseUint":
		return &Schema{Type: "integer", Format: "int64"}, true
	case "strconv.ParseBool":
		return &Schema{Type: "boolean"}, true
	case "strconv.ParseFloat":
		return &Schema{Type: "number", Format: "double"}, true
	default:
		return nil, false
	}
}

func calledFunctionName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return selectorName(typed)
	default:
		return ""
	}
}

func buildQueryParamsFromType(typeName string, currentFile *fileInfo, ctx *packageContext) []QueryParam {
	structInfo, ok := ctx.structs[typeName]
	if !ok && currentFile != nil && currentFile.pkg != nil {
		structInfo = ctx.structs[currentFile.pkg.name+"."+typeName]
	}
	if structInfo == nil {
		return nil
	}

	var params []QueryParam
	for _, field := range structInfo.structType.Fields.List {
		name, skip := extractQueryFieldName(field)
		if skip {
			continue
		}

		schema := schemaFromExpr(field.Type, structInfo.file, nil, map[string]bool{}, ctx)
		if schema == nil {
			continue
		}

		param := QueryParam{
			Name:   name,
			Type:   schema.Type,
			Format: schema.Format,
		}

		if field.Tag != nil {
			param.Required = hasBindingOption(field.Tag.Value, "required")
		}

		if refName, ok := refNameFromSchema(schema); ok {
			param.TypeName = refName
		}

		if schema.Type == "array" && schema.Items != nil {
			param.Type = "array"
		}

		params = append(params, param)
	}

	return params
}

func extractQueryFieldName(field *ast.Field) (string, bool) {
	if field.Tag != nil {
		if name, ok := extractTagValue(field.Tag.Value, "form"); ok {
			if name == "-" {
				return "", true
			}
			if name != "" {
				return name, false
			}
		}
	}

	return extractFieldName(field)
}

func refNameFromSchema(schema *Schema) (string, bool) {
	const prefix = "#/components/schemas/"
	if schema == nil || schema.Ref == "" || len(schema.Ref) <= len(prefix) || schema.Ref[:len(prefix)] != prefix {
		return "", false
	}

	return schema.Ref[len(prefix):], true
}
