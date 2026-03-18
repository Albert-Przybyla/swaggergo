package service

import (
	"testing"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
)

func TestBuildRouterSpecIncludesBodyAndQueryParams(t *testing.T) {
	t.Parallel()

	spec := buildRouterSpec(
		&config.Config{
			Info: config.InfoConfig{
				Title:   "API",
				Version: "1.0.0",
			},
		},
		config.RouterConfig{
			BasePath: "/api",
		},
		&ginparser.ParsedRouter{
			Routes: []ginparser.Route{
				{
					Method:   "GET",
					Path:     "/users/:id",
					Handler:  "ListUsers",
					BodyType: "",
					QueryParams: []ginparser.QueryParam{
						{Name: "page", Type: "integer", Format: "int64", Required: true},
						{Name: "active", Type: "boolean"},
					},
				},
				{
					Method:   "POST",
					Path:     "/users",
					Handler:  "CreateUser",
					BodyType: "CreateUserRequest",
				},
			},
			Schemas: map[string]*ginparser.Schema{
				"CreateUserRequest": {
					Type: "object",
					Properties: map[string]*ginparser.Schema{
						"name": {Type: "string"},
					},
					Required: []string{"name"},
				},
			},
		},
	)

	getOp := spec.Paths["/api/users/{id}"].Get
	if getOp == nil {
		t.Fatal("GET operation not generated")
	}

	if len(getOp.Parameters) != 3 {
		t.Fatalf("expected 3 parameters (path + 2 query), got %d", len(getOp.Parameters))
	}

	postOp := spec.Paths["/api/users"].Post
	if postOp == nil || postOp.RequestBody == nil {
		t.Fatal("POST request body not generated")
	}

	ref := postOp.RequestBody.Content["application/json"].Schema.Ref
	if ref != "#/components/schemas/CreateUserRequest" {
		t.Fatalf("unexpected request body ref: %s", ref)
	}

	if spec.Components.Schemas["CreateUserRequest"] == nil {
		t.Fatal("CreateUserRequest schema not copied into components")
	}
}
