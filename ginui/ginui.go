package ginui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html.tmpl
var templateFS embed.FS

type UI string

const (
	UISwaggerUI UI = "swagger-ui"
	UIReDoc     UI = "redoc"
	UIScalar    UI = "scalar"
)

type Config struct {
	UI      UI
	Title   string
	SpecURL string
}

func UIHandler(cfg Config) gin.HandlerFunc {
	resolved := normalizeConfig(cfg)

	return func(c *gin.Context) {
		page, err := render(resolved)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to render docs ui")
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	}
}

func RegisterUI(router gin.IRouter, route string, cfg Config) {
	route = strings.TrimSpace(route)
	if route == "" {
		route = "/docs"
	}

	handler := UIHandler(cfg)
	router.GET(route, handler)

	if strings.HasSuffix(route, "/*any") {
		return
	}

	trimmed := strings.TrimSuffix(route, "/")
	if trimmed == "" {
		trimmed = "/"
	}

	router.GET(trimmed+"/*any", handler)
}

type pageData struct {
	Title   string
	SpecURL string
}

func normalizeConfig(cfg Config) Config {
	if cfg.UI == "" {
		cfg.UI = UISwaggerUI
	}
	if cfg.Title == "" {
		cfg.Title = defaultTitle(cfg.UI)
	}
	if cfg.SpecURL == "" {
		cfg.SpecURL = "/docs/swagger.yaml"
	}

	return cfg
}

func defaultTitle(ui UI) string {
	switch ui {
	case UIReDoc:
		return "ReDoc"
	case UIScalar:
		return "Scalar API Reference"
	default:
		return "Swagger UI"
	}
}

func render(cfg Config) ([]byte, error) {
	tpl, err := pageTemplate(cfg.UI)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, pageData{
		Title:   cfg.Title,
		SpecURL: cfg.SpecURL,
	}); err != nil {
		return nil, fmt.Errorf("execute docs ui template: %w", err)
	}

	return out.Bytes(), nil
}

func pageTemplate(ui UI) (*template.Template, error) {
	switch ui {
	case UISwaggerUI:
		return template.ParseFS(templateFS, "templates/swagger-ui.html.tmpl")
	case UIReDoc:
		return template.ParseFS(templateFS, "templates/redoc.html.tmpl")
	case UIScalar:
		return template.ParseFS(templateFS, "templates/scalar.html.tmpl")
	default:
		return nil, fmt.Errorf("unsupported docs ui %q", ui)
	}
}
