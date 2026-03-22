package service

import (
	"strings"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
	"github.com/Albert-Przybyla/swaggergo/internal/generator"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
)

func buildOperation(route ginparser.Route, fullPath string, tags []config.TagConfig, defaultSecurity []string, knownSchemas map[string]*ginparser.Schema) *generator.Operation {
	op := &generator.Operation{
		OperationID: route.Method + "_" + strings.ReplaceAll(fullPath, "/", "_"),
		Summary:     route.Summary,
		Description: route.Description,
		Tags:        buildOperationTags(route, fullPath, tags),
		Responses: map[string]generator.Response{
			"200": {Description: "OK"},
		},
	}

	statusCode := route.Response.StatusCode
	if statusCode == "" {
		statusCode = "200"
	}
	if statusCode != "200" {
		delete(op.Responses, "200")
		op.Responses[statusCode] = generator.Response{Description: defaultResponseDescription(statusCode)}
	}

	if security := resolveOperationSecurity(route, fullPath, tags, defaultSecurity); security != nil {
		op.Security = security
	}

	for _, param := range extractPathParams(fullPath) {
		op.Parameters = addParam(op.Parameters, param)
	}

	for _, param := range route.QueryParams {
		op.Parameters = addParam(op.Parameters, buildQueryParameter(param))
	}

	if route.BodyType != "" {
		requestSchema := schemaForType(route.BodyType, knownSchemas)
		op.RequestBody = &generator.RequestBody{
			Required: true,
			Content: map[string]generator.MediaType{
				"application/json": {
					Schema: requestSchema,
				},
			},
		}
	}

	if route.Response.TypeName != "" {
		var schema *generator.Schema
		if route.Response.Schema != nil {
			schema = convertParsedSchema(route.Response.Schema)
		} else {
			schema = schemaForType(route.Response.TypeName, knownSchemas)
		}
		op.Responses[statusCode] = generator.Response{
			Description: defaultResponseDescription(statusCode),
			Content: map[string]generator.MediaType{
				"application/json": {
					Schema: schema,
				},
			},
		}
	} else if route.Response.Schema != nil {
		op.Responses[statusCode] = generator.Response{
			Description: defaultResponseDescription(statusCode),
			Content: map[string]generator.MediaType{
				"application/json": {
					Schema: convertParsedSchema(route.Response.Schema),
				},
			},
		}
	}

	return op
}

func schemaForType(typeName string, knownSchemas map[string]*ginparser.Schema) *generator.Schema {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return &generator.Schema{Type: "object"}
	}

	if strings.HasPrefix(typeName, "[]") {
		return &generator.Schema{
			Type:  "array",
			Items: schemaForType(strings.TrimPrefix(typeName, "[]"), knownSchemas),
		}
	}

	typeName = strings.TrimPrefix(typeName, "*")
	if builtin, ok := ginparser.BuiltinSchemaForType(typeName); ok {
		return convertParsedSchema(builtin)
	}
	if hasKnownSchema(typeName, knownSchemas) {
		return &generator.Schema{
			Ref: componentSchemaRef(typeName),
		}
	}
	return &generator.Schema{Type: "object"}
}

func hasKnownSchema(typeName string, knownSchemas map[string]*ginparser.Schema) bool {
	if len(knownSchemas) == 0 {
		return false
	}
	if _, ok := knownSchemas[typeName]; ok {
		return true
	}
	normalized := ginparser.NormalizeSchemaTypeName(typeName)
	if normalized == "" {
		return false
	}
	_, ok := knownSchemas[normalized]
	return ok
}

func schemaRefForType(typeName string) *generator.Schema {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return nil
	}

	if strings.HasPrefix(typeName, "[]") {
		return &generator.Schema{
			Type:  "array",
			Items: schemaRefForType(strings.TrimPrefix(typeName, "[]")),
		}
	}

	typeName = strings.TrimPrefix(typeName, "*")
	return &generator.Schema{
		Ref: componentSchemaRef(typeName),
	}
}

func defaultResponseDescription(statusCode string) string {
	switch statusCode {
	case "201":
		return "Created"
	case "202":
		return "Accepted"
	case "204":
		return "No Content"
	case "400":
		return "Bad Request"
	case "401":
		return "Unauthorized"
	case "403":
		return "Forbidden"
	case "404":
		return "Not Found"
	case "409":
		return "Conflict"
	case "422":
		return "Unprocessable Entity"
	case "500":
		return "Internal Server Error"
	default:
		return "OK"
	}
}

func resolveOperationSecurity(route ginparser.Route, fullPath string, tags []config.TagConfig, defaultSecurity []string) *[]generator.SecurityRequirement {
	explicit := false
	noAuth := false
	var names []string

	for _, tag := range tags {
		if !matchesTagRoute(tag, route, fullPath) {
			continue
		}

		if tag.NoAuth || containsNoAuth(tag.Security) {
			explicit = true
			noAuth = true
			names = nil
			continue
		}

		if len(tag.Security) > 0 {
			explicit = true
			noAuth = false
			names = append([]string(nil), tag.Security...)
		}
	}

	if explicit {
		if noAuth {
			empty := make([]generator.SecurityRequirement, 0)
			return &empty
		}
		return securityRequirementsPtr(buildSecurityRequirements(names))
	}

	if len(defaultSecurity) == 0 {
		return nil
	}

	return securityRequirementsPtr(buildSecurityRequirements(defaultSecurity))
}

func buildOperationTags(route ginparser.Route, fullPath string, tags []config.TagConfig) []string {
	var matched []string
	for _, tag := range tags {
		if matchesTagRoute(tag, route, fullPath) {
			matched = append(matched, tag.Name)
		}
	}

	if len(matched) > 0 {
		return matched
	}

	if route.Group == "" {
		return nil
	}

	return []string{deriveTagName(route.Group)}
}

func matchesTagRoute(tag config.TagConfig, route ginparser.Route, fullPath string) bool {
	for _, group := range tag.Groups {
		if matchRoutePattern(group, route.Group) {
			return true
		}
	}

	for _, path := range tag.Paths {
		if matchRoutePattern(path, fullPath) {
			return true
		}
	}

	return false
}

func matchRoutePattern(pattern string, value string) bool {
	pattern = normalizeMatchPath(pattern)
	value = normalizeMatchPath(value)

	if pattern == "" || value == "" {
		return false
	}

	if pattern == value {
		return true
	}

	return strings.HasPrefix(value, pattern+"/")
}

func normalizeMatchPath(path string) string {
	path = ginparser.NormalizePath("/" + strings.Trim(path, "/"))
	return strings.TrimSuffix(path, "/")
}

func deriveTagName(group string) string {
	parts := strings.Split(strings.Trim(group, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part == "" || strings.HasPrefix(part, ":") || (strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")) {
			continue
		}
		return part
	}

	return strings.Trim(group, "/")
}

func buildSecurityRequirements(names []string) []generator.SecurityRequirement {
	if len(names) == 0 {
		return nil
	}

	requirements := make([]generator.SecurityRequirement, 0, len(names))
	for _, name := range names {
		if isNoAuthSecurity(name) {
			continue
		}
		requirements = append(requirements, generator.SecurityRequirement{
			name: {},
		})
	}

	if len(requirements) == 0 {
		return nil
	}

	return requirements
}

func securityRequirementsPtr(requirements []generator.SecurityRequirement) *[]generator.SecurityRequirement {
	if requirements == nil {
		return nil
	}

	return &requirements
}

func containsNoAuth(names []string) bool {
	for _, name := range names {
		if isNoAuthSecurity(name) {
			return true
		}
	}

	return false
}

func isNoAuthSecurity(name string) bool {
	normalized := strings.TrimSpace(strings.ToLower(name))
	return normalized == "noauth" || normalized == "no_auth"
}

func buildQueryParameter(param ginparser.QueryParam) generator.Parameter {
	schema := &generator.Schema{
		Type:   param.Type,
		Format: param.Format,
	}

	if schema.Type == "" {
		schema.Type = "string"
	}

	return generator.Parameter{
		Name:     param.Name,
		In:       "query",
		Required: param.Required,
		Schema:   schema,
	}
}

func extractPathParams(path string) []generator.Parameter {
	var params []generator.Parameter

	parts := strings.Split(path, "/")
	for _, part := range parts {
		if !strings.HasPrefix(part, "{") || !strings.HasSuffix(part, "}") {
			continue
		}

		params = append(params, generator.Parameter{
			Name:     part[1 : len(part)-1],
			In:       "path",
			Required: true,
			Schema: &generator.Schema{
				Type: "string",
			},
		})
	}

	return params
}

func addParam(params []generator.Parameter, candidate generator.Parameter) []generator.Parameter {
	for _, existing := range params {
		if existing.Name == candidate.Name && existing.In == candidate.In {
			return params
		}
	}

	return append(params, candidate)
}
