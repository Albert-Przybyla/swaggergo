package service

import (
	"strings"

	"github.com/Albert-Przybyla/swaggergo/internal/generator"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
)

func buildOperation(route ginparser.Route, fullPath string) *generator.Operation {
	opID := route.Handler
	if opID == "" {
		opID = route.Method + "_" + strings.ReplaceAll(fullPath, "/", "_")
	}

	op := &generator.Operation{
		OperationID: opID,
		Tags:        buildOperationTags(route),
		Responses: map[string]generator.Response{
			"200": {Description: "OK"},
		},
	}

	for _, param := range extractPathParams(fullPath) {
		op.Parameters = addParam(op.Parameters, param)
	}

	for _, param := range route.QueryParams {
		op.Parameters = addParam(op.Parameters, buildQueryParameter(param))
	}

	if route.BodyType != "" {
		op.RequestBody = &generator.RequestBody{
			Required: true,
			Content: map[string]generator.MediaType{
				"application/json": {
					Schema: &generator.Schema{
						Ref: "#/components/schemas/" + route.BodyType,
					},
				},
			},
		}
	}

	return op
}

func buildOperationTags(route ginparser.Route) []string {
	if route.Group == "" {
		return nil
	}

	return []string{route.Group}
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
