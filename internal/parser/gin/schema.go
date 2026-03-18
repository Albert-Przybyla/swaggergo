package gin

import (
	"go/ast"
	"strings"
)

type Schema struct {
	Ref        string
	Type       string
	Format     string
	Properties map[string]*Schema
	Items      *Schema
	Required   []string
}

func collectRouteSchemas(routes []Route, ctx *packageContext) map[string]*Schema {
	schemas := make(map[string]*Schema)
	seen := make(map[string]bool)

	for _, route := range routes {
		if route.BodyType != "" {
			addSchemaFromTypeName(schemas, seen, route.BodyType, ctx)
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
	baseName := trimPointer(typeName)
	if baseName == "" || isBuiltinTypeName(baseName) {
		return
	}

	if seen != nil && seen[baseName] {
		return
	}

	info := ctx.structs[baseName]
	if info == nil {
		return
	}

	if seen != nil {
		seen[baseName] = true
	}

	if dst != nil {
		dst[baseName] = schemaFromStruct(info, dst, seen, ctx)
	}
}

func schemaFromStruct(info *structInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, field := range info.structType.Fields.List {
		fieldName, skip := extractFieldName(field)
		if skip {
			continue
		}

		fieldSchema := schemaFromExpr(field.Type, info.file, dst, seen, ctx)
		if fieldSchema == nil {
			continue
		}

		schema.Properties[fieldName] = fieldSchema
		if !isOptionalField(field) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

func schemaFromExpr(expr ast.Expr, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	switch typed := expr.(type) {
	case *ast.Ident:
		if schema, ok := builtinSchema(typed.Name); ok {
			return schema
		}

		addSchemaFromTypeName(dst, seen, typed.Name, ctx)
		return &Schema{Ref: "#/components/schemas/" + typed.Name}
	case *ast.StarExpr:
		return schemaFromExpr(typed.X, currentFile, dst, seen, ctx)
	case *ast.ArrayType:
		return &Schema{
			Type:  "array",
			Items: schemaFromExpr(typed.Elt, currentFile, dst, seen, ctx),
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

		addSchemaFromTypeName(dst, seen, fullName, ctx)
		return &Schema{Ref: "#/components/schemas/" + fullName}
	case *ast.StructType:
		return inlineSchemaFromStruct(typed, currentFile, dst, seen, ctx)
	case *ast.MapType:
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "object"}
	}
}

func inlineSchemaFromStruct(structType *ast.StructType, currentFile *fileInfo, dst map[string]*Schema, seen map[string]bool, ctx *packageContext) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, field := range structType.Fields.List {
		fieldName, skip := extractFieldName(field)
		if skip {
			continue
		}

		fieldSchema := schemaFromExpr(field.Type, currentFile, dst, seen, ctx)
		if fieldSchema == nil {
			continue
		}

		schema.Properties[fieldName] = fieldSchema
		if !isOptionalField(field) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

func builtinSchema(typeName string) (*Schema, bool) {
	switch trimPointer(typeName) {
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
		return nil, false
	}
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
