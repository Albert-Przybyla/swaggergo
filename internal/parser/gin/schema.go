package gin

import (
	"go/ast"
	"strings"
)

type Schema struct {
	Ref         string
	Type        string
	Format      string
	Description string
	Properties  map[string]*Schema
	Items       *Schema
	Required    []string
}

func collectRouteSchemas(routes []Route, ctx *packageContext) map[string]*Schema {
	schemas := make(map[string]*Schema)
	seen := make(map[string]bool)

	for _, route := range routes {
		if route.BodyType != "" {
			addSchemaFromTypeName(schemas, seen, route.BodyType, ctx)
		}
		if route.Response.TypeName != "" {
			addSchemaFromTypeName(schemas, seen, route.Response.TypeName, ctx)
		}

		for _, param := range route.QueryParams {
			if param.TypeName != "" {
				addSchemaFromTypeName(schemas, seen, param.TypeName, ctx)
			}
		}
	}

	return schemas
}

func addSchemaFromTypeName(dst map[string]*Schema, seen map[string]bool, typeName string, ctx *packageContext) {
	typeName = normalizeSchemaTypeName(typeName)
	if typeName == "" || isBuiltinTypeName(typeName) {
		return
	}

	if seen != nil && seen[typeName] {
		return
	}

	baseName, typeArgs := splitGenericTypeName(typeName)
	info := ctx.structs[baseName]
	if info == nil {
		return
	}

	if seen != nil {
		seen[typeName] = true
	}

	if dst != nil {
		if len(typeArgs) > 0 && len(info.typeParams) > 0 {
			dst[typeName] = schemaFromStructWithBindings(info, dst, seen, ctx, bindTypeParams(info.typeParams, typeArgs))
			return
		}
		dst[typeName] = schemaFromStruct(info, dst, seen, ctx)
	}
}

func normalizeSchemaTypeName(typeName string) string {
	typeName = trimPointer(typeName)
	for strings.HasPrefix(typeName, "[]") {
		typeName = strings.TrimPrefix(typeName, "[]")
		typeName = trimPointer(typeName)
	}
	return typeName
}

func NormalizeSchemaTypeName(typeName string) string {
	return normalizeSchemaTypeName(typeName)
}

func schemaFromStruct(info *structInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	return schemaFromStructWithBindings(info, dst, seen, ctx, nil)
}

func schemaFromStructWithBindings(info *structInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext, bindings map[string]string) *Schema {
	schema := &Schema{
		Type:        "object",
		Description: parseDescription(info.doc),
		Properties:  make(map[string]*Schema),
	}

	for _, field := range info.structType.Fields.List {
		fieldName, skip := extractFieldName(field)
		if skip {
			continue
		}

		fieldSchema := schemaFromExprWithBindings(field.Type, info.file, dst, seen, ctx, bindings)
		if fieldSchema == nil {
			continue
		}

		fieldSchema.Description = fieldDescription(field)
		schema.Properties[fieldName] = fieldSchema
		if !isOptionalField(field) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

func schemaFromExpr(expr ast.Expr, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	return schemaFromExprWithBindings(expr, currentFile, dst, seen, ctx, nil)
}

func schemaFromExprWithBindings(expr ast.Expr, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext, bindings map[string]string) *Schema {
	switch typed := expr.(type) {
	case *ast.Ident:
		if bindings != nil {
			if mapped := normalizeSchemaTypeName(bindings[typed.Name]); mapped != "" {
				return schemaForTypeName(mapped, currentFile, dst, seen, ctx)
			}
		}
		if schema, ok := builtinSchema(typed.Name); ok {
			return schema
		}

		return schemaForTypeName(typed.Name, currentFile, dst, seen, ctx)
	case *ast.StarExpr:
		return schemaFromExprWithBindings(typed.X, currentFile, dst, seen, ctx, bindings)
	case *ast.ArrayType:
		return &Schema{
			Type:  "array",
			Items: schemaFromExprWithBindings(typed.Elt, currentFile, dst, seen, ctx, bindings),
		}
	case *ast.SelectorExpr:
		fullName := selectorName(typed)
		if currentFile != nil {
			if left, ok := typed.X.(*ast.Ident); ok {
				if importPath, ok := currentFile.imports[left.Name]; ok {
					if pkg := ctx.packages[importPath]; pkg != nil {
						fullName = pkg.name + "." + typed.Sel.Name
					}
				}
			}
		}

		if schema, ok := builtinSchema(fullName); ok {
			return schema
		}

		return schemaForTypeName(fullName, currentFile, dst, seen, ctx)
	case *ast.IndexExpr:
		typeName := exprTypeNameForFile(typed, currentFile)
		return schemaForTypeName(typeName, currentFile, dst, seen, ctx)
	case *ast.IndexListExpr:
		typeName := exprTypeNameForFile(typed, currentFile)
		return schemaForTypeName(typeName, currentFile, dst, seen, ctx)
	case *ast.StructType:
		return inlineSchemaFromStructWithBindings(typed, currentFile, dst, seen, ctx, bindings)
	case *ast.MapType:
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "object"}
	}
}

func inlineSchemaFromStruct(structType *ast.StructType, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	return inlineSchemaFromStructWithBindings(structType, currentFile, dst, seen, ctx, nil)
}

func inlineSchemaFromStructWithBindings(structType *ast.StructType, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext, bindings map[string]string) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, field := range structType.Fields.List {
		fieldName, skip := extractFieldName(field)
		if skip {
			continue
		}

		fieldSchema := schemaFromExprWithBindings(field.Type, currentFile, dst, seen, ctx, bindings)
		if fieldSchema == nil {
			continue
		}

		fieldSchema.Description = fieldDescription(field)
		schema.Properties[fieldName] = fieldSchema
		if !isOptionalField(field) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

func schemaForTypeName(typeName string, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	typeName = normalizeSchemaTypeName(typeName)
	if typeName == "" {
		return &Schema{Type: "object"}
	}

	if schema, ok := builtinSchema(typeName); ok {
		return schema
	}

	addSchemaFromTypeName(dst, seen, typeName, ctx)
	return &Schema{Ref: "#/components/schemas/" + typeName}
}

func bindTypeParams(names []string, args []string) map[string]string {
	if len(names) == 0 || len(args) == 0 {
		return nil
	}

	bindings := make(map[string]string, len(names))
	for i, name := range names {
		if i >= len(args) {
			break
		}
		bindings[name] = args[i]
	}
	return bindings
}

func splitGenericTypeName(typeName string) (string, []string) {
	typeName = normalizeSchemaTypeName(typeName)
	start := strings.Index(typeName, "[")
	end := strings.LastIndex(typeName, "]")
	if start == -1 || end == -1 || end < start {
		return typeName, nil
	}

	base := strings.TrimSpace(typeName[:start])
	rawArgs := typeName[start+1 : end]
	return base, splitTopLevel(rawArgs, ',')
}

func splitTopLevel(input string, sep rune) []string {
	var parts []string
	depth := 0
	start := 0

	for i, r := range input {
		switch r {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if r == sep && depth == 0 {
				part := strings.TrimSpace(input[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}

	last := strings.TrimSpace(input[start:])
	if last != "" {
		parts = append(parts, last)
	}

	return parts
}

func builtinSchema(typeName string) (*Schema, bool) {
	typeName = trimPointer(typeName)
	switch typeName {
	case "string":
		return &Schema{Type: "string"}, true
	case "bool":
		return &Schema{Type: "boolean"}, true
	case "int", "int8", "int16", "int32":
		return &Schema{Type: "integer", Format: "int32"}, true
	case "int64":
		return &Schema{Type: "integer", Format: "int64"}, true
	case "uint", "uint8", "uint16", "uint32":
		return &Schema{Type: "integer", Format: "int32"}, true
	case "uint64":
		return &Schema{Type: "integer", Format: "int64"}, true
	case "float32":
		return &Schema{Type: "number", Format: "float"}, true
	case "float64":
		return &Schema{Type: "number", Format: "double"}, true
	case "time.Time":
		return &Schema{Type: "string", Format: "date-time"}, true
	default:
		if typeName == "UUID" || strings.HasSuffix(typeName, ".UUID") {
			return &Schema{Type: "string", Format: "uuid"}, true
		}
		return nil, false
	}
}

func BuiltinSchemaForType(typeName string) (*Schema, bool) {
	return builtinSchema(typeName)
}

func extractFieldName(field *ast.Field) (string, bool) {
	if field.Tag != nil {
		for _, key := range []string{"json", "form"} {
			if name, ok := extractTagValue(field.Tag.Value, key); ok {
				if name == "-" {
					return "", true
				}
				if name != "" {
					return name, false
				}
			}
		}
	}

	if len(field.Names) == 0 {
		return "", true
	}

	return field.Names[0].Name, false
}

func isOptionalField(field *ast.Field) bool {
	if _, ok := field.Type.(*ast.StarExpr); ok {
		return true
	}

	if field.Tag == nil {
		return false
	}

	for _, key := range []string{"json", "form"} {
		if raw, ok := extractFullTagValue(field.Tag.Value, key); ok && strings.Contains(raw, "omitempty") {
			return true
		}
	}

	return hasBindingOption(field.Tag.Value, "omitempty")
}

func isBuiltinTypeName(typeName string) bool {
	_, ok := builtinSchema(typeName)
	return ok
}

func fieldDescription(field *ast.Field) string {
	if field == nil {
		return ""
	}

	if description := parseDescription(field.Doc); description != "" {
		return description
	}

	return parseDescription(field.Comment)
}
