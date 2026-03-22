package service

import (
	"strings"
	"unicode"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
	"github.com/Albert-Przybyla/swaggergo/internal/generator"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
)

func newComponents() *generator.Components {
	return &generator.Components{
		Schemas:         map[string]*generator.Schema{},
		SecuritySchemes: map[string]generator.SecurityScheme{},
		Parameters:      map[string]generator.Parameter{},
		Responses:       map[string]generator.Response{},
		RequestBodies:   map[string]generator.RequestBody{},
		Headers:         map[string]generator.Header{},
	}
}

func applyConfiguredComponents(dst *generator.Components, src *config.ComponentsConfig) {
	if src == nil {
		return
	}

	for name, schema := range src.Schemas {
		dst.Schemas[name] = convertSchema(schema)
	}

	for name, scheme := range src.SecuritySchemes {
		dst.SecuritySchemes[name] = generator.SecurityScheme{
			Type:         scheme.Type,
			Scheme:       scheme.Scheme,
			BearerFormat: scheme.BearerFormat,
			In:           scheme.In,
			Name:         scheme.Name,
			Description:  scheme.Description,
		}
	}
}

func applyParsedSchemas(dst *generator.Components, schemas map[string]*ginparser.Schema) {
	for name, schema := range schemas {
		dst.Schemas[componentSchemaName(name)] = convertParsedSchema(schema)
	}
}

func convertSchema(v interface{}) *generator.Schema {
	return &generator.Schema{
		Type: "object",
	}
}

func convertParsedSchema(schema *ginparser.Schema) *generator.Schema {
	if schema == nil {
		return nil
	}

	converted := &generator.Schema{
		Ref:         sanitizeSchemaRef(schema.Ref),
		Type:        schema.Type,
		Format:      schema.Format,
		Description: schema.Description,
		Required:    schema.Required,
	}

	if schema.Items != nil {
		converted.Items = convertParsedSchema(schema.Items)
	}

	if len(schema.Properties) > 0 {
		converted.Properties = make(map[string]*generator.Schema, len(schema.Properties))
		for name, property := range schema.Properties {
			converted.Properties[name] = convertParsedSchema(property)
		}
	}

	return converted
}

func componentSchemaRef(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ""
	}

	return "#/components/schemas/" + componentSchemaName(typeName)
}

func sanitizeSchemaRef(ref string) string {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return ref
	}

	return prefix + componentSchemaName(strings.TrimPrefix(ref, prefix))
}

func componentSchemaName(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ""
	}

	var b strings.Builder
	lastUnderscore := false

	for _, r := range typeName {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}

		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}

	name := strings.Trim(b.String(), "_")
	if name == "" {
		return "Schema"
	}

	return name
}
