package gin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileExtractsBodyAndQueryParams(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	routerFile := filepath.Join(dir, "router.go")
	handlerFile := filepath.Join(dir, "handlers.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, h *UserHandler) {
	users := r.Group("/users")
	users.GET("/:id", h.List)
	users.POST("", h.Create)
}
`)

	writeTestFile(t, handlerFile, `package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

type ListFilters struct {
	Page   int    `+"`form:\"page\" binding:\"required\"`"+`
	Search string `+"`form:\"search\"`"+`
	Active bool   `+"`form:\"active\"`"+`
}

type CreateUserRequest struct {
	Name    string `+"`json:\"name\"`"+`
	Age     int    `+"`json:\"age\"`"+`
	Enabled bool   `+"`json:\"enabled,omitempty\"`"+`
}

func (h *UserHandler) List(c *gin.Context) {
	sort := c.Query("sort")
	_ = sort

	pageRaw := c.Query("page")
	page, _ := strconv.Atoi(pageRaw)
	_ = page

	active, _ := strconv.ParseBool(c.Query("active"))
	_ = active

	var filters ListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		return
	}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if got, want := len(parsed.Routes), 2; got != want {
		t.Fatalf("expected %d routes, got %d", want, got)
	}

	getRoute := routeByMethod(parsed.Routes, "GET")
	if getRoute == nil {
		t.Fatal("GET route not found")
	}

	assertQueryParam(t, getRoute.QueryParams, QueryParam{Name: "sort", Type: "string"})
	assertQueryParam(t, getRoute.QueryParams, QueryParam{Name: "page", Type: "integer", Format: "int32", Required: true})
	assertQueryParam(t, getRoute.QueryParams, QueryParam{Name: "active", Type: "boolean"})
	assertQueryParam(t, getRoute.QueryParams, QueryParam{Name: "search", Type: "string"})

	postRoute := routeByMethod(parsed.Routes, "POST")
	if postRoute == nil {
		t.Fatal("POST route not found")
	}

	if got, want := postRoute.BodyType, "CreateUserRequest"; got != want {
		t.Fatalf("expected body type %q, got %q", want, got)
	}

	bodySchema := parsed.Schemas["CreateUserRequest"]
	if bodySchema == nil {
		t.Fatal("CreateUserRequest schema not found")
	}

	if bodySchema.Properties["name"] == nil || bodySchema.Properties["name"].Type != "string" {
		t.Fatalf("expected name field to be string, got %#v", bodySchema.Properties["name"])
	}

	if bodySchema.Properties["age"] == nil || bodySchema.Properties["age"].Type != "integer" {
		t.Fatalf("expected age field to be integer, got %#v", bodySchema.Properties["age"])
	}

	if !contains(bodySchema.Required, "name") || !contains(bodySchema.Required, "age") {
		t.Fatalf("expected required fields to include name and age, got %#v", bodySchema.Required)
	}

	if contains(bodySchema.Required, "enabled") {
		t.Fatalf("enabled should not be required: %#v", bodySchema.Required)
	}
}

func TestParseFileExtractsSchemasFromOtherProjectPackages(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	apiDir := filepath.Join(root, "internal", "api")
	modelsDir := filepath.Join(root, "models")

	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("MkdirAll(apiDir) error = %v", err)
	}
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("MkdirAll(modelsDir) error = %v", err)
	}

	routerFile := filepath.Join(apiDir, "router.go")
	handlerFile := filepath.Join(apiDir, "handlers.go")
	modelFile := filepath.Join(modelsDir, "user.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, h *UserHandler) {
	users := r.Group("/users")
	users.PATCH("/password", h.ChangePassword)
}
`)

	writeTestFile(t, handlerFile, `package api

import (
	"github.com/gin-gonic/gin"
	"example.com/app/models"
)

type UserHandler struct{}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var payload models.ChangePasswordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		return
	}
}
`)

	writeTestFile(t, modelFile, `package models

type ChangePasswordPayload struct {
	CurrentPassword string `+"`json:\"currentPassword\"`"+`
	NewPassword     string `+"`json:\"newPassword\"`"+`
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{
		ProjectRoot: root,
		ModulePath:  "example.com/app",
	})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	route := routeByMethod(parsed.Routes, "PATCH")
	if route == nil {
		t.Fatal("PATCH route not found")
	}

	if got, want := route.BodyType, "models.ChangePasswordPayload"; got != want {
		t.Fatalf("expected body type %q, got %q", want, got)
	}

	schema := parsed.Schemas["models.ChangePasswordPayload"]
	if schema == nil {
		t.Fatal("models.ChangePasswordPayload schema not found")
	}

	if schema.Properties["currentPassword"] == nil || schema.Properties["currentPassword"].Type != "string" {
		t.Fatalf("expected currentPassword field to be string, got %#v", schema.Properties["currentPassword"])
	}

	if schema.Properties["newPassword"] == nil || schema.Properties["newPassword"].Type != "string" {
		t.Fatalf("expected newPassword field to be string, got %#v", schema.Properties["newPassword"])
	}
}

func TestParseFileExtractsDescriptionsAndHttpxBody(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	handlerDir := filepath.Join(root, "handler")
	dtoDir := filepath.Join(root, "dto")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("MkdirAll(appDir) error = %v", err)
	}
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		t.Fatalf("MkdirAll(handlerDir) error = %v", err)
	}
	if err := os.MkdirAll(dtoDir, 0755); err != nil {
		t.Fatalf("MkdirAll(dtoDir) error = %v", err)
	}

	routerFile := filepath.Join(appDir, "router.go")
	handlerFile := filepath.Join(handlerDir, "preferences.go")
	dtoFile := filepath.Join(dtoDir, "preferences.go")

	writeTestFile(t, routerFile, `package app

import (
	"example.com/app/handler"
	"github.com/gin-gonic/gin"
)

func ProvideRouter(preferencesHandler *handler.PreferencesHandler) *gin.Engine {
	r := gin.Default()
	api := r.Group("api/v1")
	preferences := api.Group("preferences/:key")
	preferences.POST("", preferencesHandler.Create)
	return r
}
`)

	writeTestFile(t, handlerFile, `package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/example/common/httpx"
	"example.com/app/dto"
)

type PreferencesHandler struct{}

// @Title Create Preferences
// @Description Creates preference for specific key.
func (h *PreferencesHandler) Create(c *gin.Context) {
	var body dto.PreferencesRequest
	if err := httpx.ValidateStruct(c, &body); err != nil {
		return
	}
}
`)

	writeTestFile(t, dtoFile, `package dto

// PreferencesRequest payload for preference operations.
type PreferencesRequest struct {
	// Human readable preference name.
	Name string `+"`json:\"name\" validate:\"required\"`"+`
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{
		ProjectRoot: root,
		ModulePath:  "example.com/app",
	})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	route := routeByMethod(parsed.Routes, "POST")
	if route == nil {
		t.Fatal("POST route not found")
	}

	if got, want := route.Summary, "Create Preferences"; got != want {
		t.Fatalf("expected summary %q, got %q", want, got)
	}

	if got, want := route.Description, "Creates preference for specific key."; got != want {
		t.Fatalf("expected description %q, got %q", want, got)
	}

	if got, want := route.BodyType, "dto.PreferencesRequest"; got != want {
		t.Fatalf("expected body type %q, got %q", want, got)
	}

	schema := parsed.Schemas["dto.PreferencesRequest"]
	if schema == nil {
		t.Fatal("dto.PreferencesRequest schema not found")
	}

	if got, want := schema.Description, "PreferencesRequest payload for preference operations."; got != want {
		t.Fatalf("expected schema description %q, got %q", want, got)
	}

	if schema.Properties["name"] == nil || schema.Properties["name"].Description != "Human readable preference name." {
		t.Fatalf("expected field description to be preserved, got %#v", schema.Properties["name"])
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func routeByMethod(routes []Route, method string) *Route {
	for i := range routes {
		if routes[i].Method == method {
			return &routes[i]
		}
	}

	return nil
}

func assertQueryParam(t *testing.T, params []QueryParam, want QueryParam) {
	t.Helper()

	for _, param := range params {
		if param.Name != want.Name {
			continue
		}

		if param.Type != want.Type || param.Format != want.Format || param.Required != want.Required {
			t.Fatalf("unexpected query param for %q: got %#v want %#v", want.Name, param, want)
		}

		return
	}

	t.Fatalf("query param %q not found in %#v", want.Name, params)
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}

	return false
}
