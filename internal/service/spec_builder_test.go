package service

import (
	"strings"
	"testing"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
	"gopkg.in/yaml.v3"
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
			BasePath:        "/api",
			DefaultSecurity: []string{"BearerAuth"},
			Tags: []config.TagConfig{
				{
					Name:        "users",
					Description: "Users operations",
					Groups:      []string{"/users"},
				},
			},
			SecuritySchemes: map[string]config.SecuritySchemeConfig{
				"BearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
			},
		},
		&ginparser.ParsedRouter{
			Routes: []ginparser.Route{
				{
					Method:      "GET",
					Group:       "/users",
					Path:        "/:id",
					Handler:     "ListUsers",
					Summary:     "List users",
					Description: "Returns users with filters.",
					BodyType:    "",
					QueryParams: []ginparser.QueryParam{
						{Name: "page", Type: "integer", Format: "int64", Required: true},
						{Name: "active", Type: "boolean"},
					},
				},
				{
					Method:   "POST",
					Group:    "/users",
					Path:     "",
					Handler:  "CreateUser",
					BodyType: "CreateUserRequest",
				},
			},
			Schemas: map[string]*ginparser.Schema{
				"CreateUserRequest": {
					Type:        "object",
					Description: "Payload used to create a user.",
					Properties: map[string]*ginparser.Schema{
						"name": {
							Type:        "string",
							Description: "User display name.",
						},
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

	if got, want := getOp.Summary, "List users"; got != want {
		t.Fatalf("expected summary %q, got %q", want, got)
	}

	if got, want := getOp.Description, "Returns users with filters."; got != want {
		t.Fatalf("expected description %q, got %q", want, got)
	}

	if len(getOp.Tags) != 1 || getOp.Tags[0] != "users" {
		t.Fatalf("expected configured tag to be applied, got %#v", getOp.Tags)
	}

	if getOp.Security == nil || len(*getOp.Security) != 1 {
		t.Fatalf("expected operation security to inherit default security, got %#v", getOp.Security)
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

	if got, want := spec.Components.Schemas["CreateUserRequest"].Description, "Payload used to create a user."; got != want {
		t.Fatalf("expected schema description %q, got %q", want, got)
	}

	if got, want := spec.Components.Schemas["CreateUserRequest"].Properties["name"].Description, "User display name."; got != want {
		t.Fatalf("expected field description %q, got %q", want, got)
	}

	if len(spec.Security) != 1 {
		t.Fatalf("expected one default security requirement, got %#v", spec.Security)
	}
}

func TestBuildRouterSpecAllowsSecurityOverrideAndNoAuth(t *testing.T) {
	t.Parallel()

	spec := buildRouterSpec(
		&config.Config{
			Info: config.InfoConfig{
				Title:   "API",
				Version: "1.0.0",
			},
		},
		config.RouterConfig{
			BasePath:        "/api",
			DefaultSecurity: []string{"BearerAuth"},
			Tags: []config.TagConfig{
				{
					Name:     "public",
					Paths:    []string{"/api/health"},
					Security: []string{"noAuth", "ApiKeyAuth"},
				},
				{
					Name:     "admin",
					Groups:   []string{"/admin"},
					Security: []string{"ApiKeyAuth"},
				},
			},
			SecuritySchemes: map[string]config.SecuritySchemeConfig{
				"BearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
				"ApiKeyAuth": {
					Type: "apiKey",
					In:   "header",
					Name: "X-API-Key",
				},
			},
		},
		&ginparser.ParsedRouter{
			Routes: []ginparser.Route{
				{
					Method:  "GET",
					Group:   "/users",
					Path:    "",
					Handler: "UsersList",
				},
				{
					Method:  "GET",
					Group:   "/admin",
					Path:    "/stats",
					Handler: "AdminStats",
				},
				{
					Method:  "GET",
					Group:   "/health",
					Path:    "",
					Handler: "Health",
				},
			},
		},
	)

	usersOp := spec.Paths["/api/users"].Get
	if usersOp == nil || usersOp.Security == nil || len(*usersOp.Security) != 1 {
		t.Fatalf("expected default security on users route, got %#v", usersOp)
	}

	adminOp := spec.Paths["/api/admin/stats"].Get
	if adminOp == nil || adminOp.Security == nil || len(*adminOp.Security) != 1 {
		t.Fatalf("expected explicit security on admin route, got %#v", adminOp)
	}
	if _, ok := (*adminOp.Security)[0]["ApiKeyAuth"]; !ok {
		t.Fatalf("expected ApiKeyAuth on admin route, got %#v", adminOp.Security)
	}

	healthOp := spec.Paths["/api/health"].Get
	if healthOp == nil {
		t.Fatal("health operation not generated")
	}
	if healthOp.Security == nil {
		t.Fatalf("expected explicit no-auth override, got nil security")
	}
	if len(*healthOp.Security) != 0 {
		t.Fatalf("expected no auth on health route, got %#v", healthOp.Security)
	}

	out, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	if !strings.Contains(string(out), "security: []") {
		t.Fatalf("expected serialized spec to contain explicit empty security array, got:\n%s", string(out))
	}
}
