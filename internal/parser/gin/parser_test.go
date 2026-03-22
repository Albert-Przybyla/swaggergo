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

func TestParseFileTreatsUUIDTypesAsStrings(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	dtoDir := filepath.Join(root, "dto")

	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("MkdirAll(apiDir) error = %v", err)
	}
	if err := os.MkdirAll(dtoDir, 0755); err != nil {
		t.Fatalf("MkdirAll(dtoDir) error = %v", err)
	}

	routerFile := filepath.Join(apiDir, "router.go")
	handlerFile := filepath.Join(apiDir, "handler.go")
	dtoFile := filepath.Join(dtoDir, "request.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, h *BundleHandler) {
	bundles := r.Group("/bundles")
	bundles.POST("", h.Create)
}
`)

	writeTestFile(t, handlerFile, `package api

import (
	"github.com/gin-gonic/gin"
	"example.com/app/dto"
)

type BundleHandler struct{}

func (h *BundleHandler) Create(c *gin.Context) {
	var req dto.CreateBundleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
}
`)

	writeTestFile(t, dtoFile, `package dto

import "github.com/google/uuid"

type BundleSizeRequest struct {
	Size string `+"`json:\"size\"`"+`
}

type CreateBundleRequest struct {
	Name        string              `+"`json:\"name\"`"+`
	Features    []uuid.UUID         `+"`json:\"features\"`"+`
	BundleSizes []BundleSizeRequest `+"`json:\"bundle_sizes\"`"+`
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{
		ProjectRoot: root,
		ModulePath:  "example.com/app",
	})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	schema := parsed.Schemas["dto.CreateBundleRequest"]
	if schema == nil {
		t.Fatal("dto.CreateBundleRequest schema not found")
	}

	features := schema.Properties["features"]
	if features == nil || features.Type != "array" || features.Items == nil {
		t.Fatalf("expected features array schema, got %#v", features)
	}
	if got, want := features.Items.Type, "string"; got != want {
		t.Fatalf("expected UUID array items to be string, got %q", got)
	}
	if got, want := features.Items.Format, "uuid"; got != want {
		t.Fatalf("expected UUID array items format %q, got %q", want, got)
	}
	if features.Items.Ref != "" {
		t.Fatalf("expected UUID array items to avoid refs, got %q", features.Items.Ref)
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

func TestParseFileResolvesNestedGroupAliases(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	routerFile := filepath.Join(dir, "router.go")
	handlerFile := filepath.Join(dir, "handlers.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, h *OrderHandler) {
	api := r.Group("api/v1")
	order := api.Group("order")
	orderAdminView := order.Group("")
	orderAdminView.GET(":id/payment", h.FetchOrderPayment)
}
`)

	writeTestFile(t, handlerFile, `package api

import "github.com/gin-gonic/gin"

type OrderHandler struct{}

func (h *OrderHandler) FetchOrderPayment(c *gin.Context) {}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	route := routeByMethod(parsed.Routes, "GET")
	if route == nil {
		t.Fatal("GET route not found")
	}

	if got, want := route.Group, "api/v1/order"; got != want {
		t.Fatalf("expected nested group path %q, got %q", want, got)
	}
	if got, want := BuildFullPath("", route.Group, route.Path), "/api/v1/order/:id/payment"; got != want {
		t.Fatalf("expected full path %q, got %q", want, got)
	}
}

func TestParseFileExtractsResponseBodyFromServiceCall(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	dtoDir := filepath.Join(root, "dto")

	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("MkdirAll(apiDir) error = %v", err)
	}
	if err := os.MkdirAll(dtoDir, 0755); err != nil {
		t.Fatalf("MkdirAll(dtoDir) error = %v", err)
	}

	routerFile := filepath.Join(apiDir, "router.go")
	handlerFile := filepath.Join(apiDir, "handler.go")
	serviceFile := filepath.Join(apiDir, "service.go")
	dtoFile := filepath.Join(dtoDir, "preferences.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, h *PreferencesHandler) {
	preferences := r.Group("/preferences/:key")
	preferences.GET("", h.List)
	preferences.POST("", h.Create)
}
`)

	writeTestFile(t, handlerFile, `package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PreferencesHandler struct {
	service *PreferencesService
}

func (h *PreferencesHandler) Create(c *gin.Context) {
	c.JSON(http.StatusCreated, StatusResponse{Status: "OK"})
}

func (h *PreferencesHandler) List(c *gin.Context) {
	res, err := h.service.List()
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, res)
}
`)

	writeTestFile(t, serviceFile, `package api

import "example.com/app/dto"

type PreferencesService struct{}

type StatusResponse struct {
	Status string `+"`json:\"status\"`"+`
}

func (s *PreferencesService) List() ([]*dto.PreferencesResponse, error) {
	response := make([]*dto.PreferencesResponse, 0, 1)
	response = append(response, &dto.PreferencesResponse{Name: "email"})
	return response, nil
}
`)

	writeTestFile(t, dtoFile, `package dto

type PreferencesResponse struct {
	Name string `+"`json:\"name\"`"+`
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{
		ProjectRoot: root,
		ModulePath:  "example.com/app",
	})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	getRoute := routeByMethod(parsed.Routes, "GET")
	if getRoute == nil {
		t.Fatal("GET route not found")
	}

	if got, want := getRoute.Response.StatusCode, "200"; got != want {
		t.Fatalf("expected GET response status %q, got %q", want, got)
	}
	if got, want := getRoute.Response.TypeName, "[]*dto.PreferencesResponse"; got != want {
		t.Fatalf("expected GET response type %q, got %q", want, got)
	}

	postRoute := routeByMethod(parsed.Routes, "POST")
	if postRoute == nil {
		t.Fatal("POST route not found")
	}

	if got, want := postRoute.Response.StatusCode, "201"; got != want {
		t.Fatalf("expected POST response status %q, got %q", want, got)
	}
	if got, want := postRoute.Response.TypeName, "StatusResponse"; got != want {
		t.Fatalf("expected POST response type %q, got %q", want, got)
	}

	if parsed.Schemas["dto.PreferencesResponse"] == nil {
		t.Fatal("dto.PreferencesResponse schema not found")
	}
	if parsed.Schemas["StatusResponse"] == nil {
		t.Fatal("StatusResponse schema not found")
	}
}

func TestParseFileExtractsGenericResponseBodyFromServiceCall(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	modelsDir := filepath.Join(root, "models")

	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("MkdirAll(apiDir) error = %v", err)
	}
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("MkdirAll(modelsDir) error = %v", err)
	}

	routerFile := filepath.Join(apiDir, "router.go")
	handlerFile := filepath.Join(apiDir, "handler.go")
	dbFile := filepath.Join(apiDir, "db.go")
	modelsFile := filepath.Join(modelsDir, "models.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, a *Api) {
	users := r.Group("/users")
	users.GET("/subusers", a.fetchAllIspSubUsers)
}
`)

	writeTestFile(t, handlerFile, `package api

import (
	"github.com/gin-gonic/gin"
	"example.com/app/models"
)

type Api struct {
	db *Postgres
}

func (a *Api) fetchAllIspSubUsers(c *gin.Context) {
	ispUsers, ispUsersErr := a.db.FetchAllIspUsers(10, 1, "")
	if ispUsersErr != nil {
		c.JSON(500, &models.ApiErrorResponse{})
		return
	}

	c.JSON(200, ispUsers)
}
`)

	writeTestFile(t, dbFile, `package api

import "example.com/app/models"

type Postgres struct{}

func (p *Postgres) FetchAllIspUsers(pageSize, pageNumber int, filter string) (*models.PaginatedResponse[models.IspUserAdminResponse], error) {
	return &models.PaginatedResponse[models.IspUserAdminResponse]{}, nil
}
`)

	writeTestFile(t, modelsFile, `package models

type ApiErrorResponse struct {
	Message string `+"`json:\"message\"`"+`
}

type PaginatedResponse[T any] struct {
	Data       []T `+"`json:\"data\"`"+`
	TotalPages int `+"`json:\"total_pages\"`"+`
	Page       int `+"`json:\"page\"`"+`
}

type IspUserAdminResponse struct {
	Username string `+"`json:\"username\"`"+`
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{
		ProjectRoot: root,
		ModulePath:  "example.com/app",
	})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	getRoute := routeByMethod(parsed.Routes, "GET")
	if getRoute == nil {
		t.Fatal("GET route not found")
	}

	if got, want := getRoute.Response.TypeName, "models.PaginatedResponse[models.IspUserAdminResponse]"; got != want {
		t.Fatalf("expected response type %q, got %q", want, got)
	}

	schema := parsed.Schemas["models.PaginatedResponse[models.IspUserAdminResponse]"]
	if schema == nil {
		t.Fatal("generic paginated schema not found")
	}

	data := schema.Properties["data"]
	if data == nil || data.Type != "array" || data.Items == nil || data.Items.Ref != "#/components/schemas/models.IspUserAdminResponse" {
		t.Fatalf("expected data field to resolve generic item type, got %#v", data)
	}
}

func TestParseFileBuildsInlineSchemaForExternalCompositeLiteralResponse(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	apiDir := filepath.Join(root, "api")

	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("MkdirAll(apiDir) error = %v", err)
	}

	routerFile := filepath.Join(apiDir, "router.go")
	handlerFile := filepath.Join(apiDir, "handler.go")

	writeTestFile(t, routerFile, `package api

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.RouterGroup, h *PreferencesHandler) {
	preferences := r.Group("/preferences/:key")
	preferences.POST("", h.Create)
}
`)

	writeTestFile(t, handlerFile, `package api

import (
	"net/http"

	"example.com/external/httpx"
	"github.com/gin-gonic/gin"
)

type PreferencesHandler struct{}

func (h *PreferencesHandler) Create(c *gin.Context) {
	c.JSON(http.StatusCreated, httpx.StatusResponse{Status: "OK"})
}
`)

	parsed, err := ParseFile(routerFile, ParseOptions{
		ProjectRoot: root,
		ModulePath:  "example.com/app",
	})
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	postRoute := routeByMethod(parsed.Routes, "POST")
	if postRoute == nil {
		t.Fatal("POST route not found")
	}

	if got, want := postRoute.Response.TypeName, "httpx.StatusResponse"; got != want {
		t.Fatalf("expected response type %q, got %q", want, got)
	}
	if postRoute.Response.Schema == nil {
		t.Fatal("expected inline schema for external composite literal")
	}
	if postRoute.Response.Schema.Properties["Status"] == nil || postRoute.Response.Schema.Properties["Status"].Type != "string" {
		t.Fatalf("expected inline schema to include Status string, got %#v", postRoute.Response.Schema)
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
