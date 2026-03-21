package ginui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUIHandlerSwaggerUI(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/swagger/*any", UIHandler(Config{
		UI:      UISwaggerUI,
		Title:   "Docs",
		SpecURL: "/docs/swagger.yaml",
	}))

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "SwaggerUIBundle") {
		t.Fatalf("expected swagger ui bundle in body, got %q", body)
	}
	if !strings.Contains(body, `url: "\/docs\/swagger.yaml"`) {
		t.Fatalf("expected spec url in body, got %q", body)
	}
}

func TestRegisterUIRegistersConvenienceRoutes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterUI(router, "/docs", Config{
		UI:      UIReDoc,
		SpecURL: "/openapi.yaml",
	})

	for _, path := range []string{"/docs", "/docs/", "/docs/index.html", "/docs/anything"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `<redoc spec-url="/openapi.yaml"></redoc>`) {
			t.Fatalf("expected redoc html for %s, got %q", path, rec.Body.String())
		}
	}
}

func TestUIHandlerScalar(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/reference", UIHandler(Config{
		UI:      UIScalar,
		SpecURL: "/docs/swagger.yaml",
	}))

	req := httptest.NewRequest(http.MethodGet, "/reference", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "@scalar/api-reference") {
		t.Fatalf("expected scalar script in body, got %q", body)
	}
}
