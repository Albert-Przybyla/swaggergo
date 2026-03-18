package service

import (
	"strings"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
	"github.com/Albert-Przybyla/swaggergo/internal/generator"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
)

func buildRouterSpec(cfg *config.Config, routerCfg config.RouterConfig, parsed *ginparser.ParsedRouter) *generator.OpenAPISpec {
	info := cfg.Info
	if routerCfg.Info != nil {
		info = *routerCfg.Info
	}

	spec := &generator.OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: generator.InfoObject{
			Title:       info.Title,
			Description: info.Description,
			Version:     info.Version,
		},
		Paths: make(map[string]*generator.PathItem),
		Components: newComponents(),
	}

	applyInfoMetadata(&spec.Info, info)
	applyServers(spec, cfg.Servers)
	applyTags(spec, routerCfg.Tags)
	applyConfiguredComponents(spec.Components, cfg.Components)
	applyConfiguredComponents(spec.Components, routerCfg.Components)
	applySecurity(spec, routerCfg.SecuritySchemes)
	applyParsedSchemas(spec.Components, parsed.Schemas)
	applyRoutes(spec, routerCfg.BasePath, parsed.Routes)

	return spec
}

func applyInfoMetadata(dst *generator.InfoObject, info config.InfoConfig) {
	if info.Contact != nil {
		dst.Contact = &generator.ContactObject{
			Name:  info.Contact.Name,
			URL:   info.Contact.URL,
			Email: info.Contact.Email,
		}
	}

	if info.License != nil {
		dst.License = &generator.LicenseObject{
			Name: info.License.Name,
			URL:  info.License.URL,
		}
	}
}

func applyServers(spec *generator.OpenAPISpec, servers []config.ServerConfig) {
	for _, server := range servers {
		spec.Servers = append(spec.Servers, generator.ServerObject{
			URL:         server.URL,
			Description: server.Description,
		})
	}
}

func applyTags(spec *generator.OpenAPISpec, tags []config.TagConfig) {
	for _, tag := range tags {
		spec.Tags = append(spec.Tags, generator.TagObject{
			Name:        tag.Name,
			Description: tag.Description,
		})
	}
}

func applySecurity(spec *generator.OpenAPISpec, schemes map[string]config.SecuritySchemeConfig) {
	for name, scheme := range schemes {
		spec.Components.SecuritySchemes[name] = generator.SecurityScheme{
			Type:         scheme.Type,
			Scheme:       scheme.Scheme,
			BearerFormat: scheme.BearerFormat,
			In:           scheme.In,
			Name:         scheme.Name,
			Description:  scheme.Description,
		}

		spec.Security = append(spec.Security, generator.SecurityRequirement{
			name: {},
		})
	}
}

func applyRoutes(spec *generator.OpenAPISpec, basePath string, routes []ginparser.Route) {
	for _, route := range routes {
		fullPath := ginparser.NormalizePath(ginparser.BuildFullPath(basePath, route.Group, route.Path))
		item := ensurePathItem(spec.Paths, fullPath)
		op := buildOperation(route, fullPath)

		switch strings.ToUpper(route.Method) {
		case "GET":
			item.Get = op
		case "POST":
			item.Post = op
		case "PUT":
			item.Put = op
		case "PATCH":
			item.Patch = op
		case "DELETE":
			item.Delete = op
		}
	}
}

func ensurePathItem(paths map[string]*generator.PathItem, path string) *generator.PathItem {
	item := paths[path]
	if item == nil {
		item = &generator.PathItem{}
		paths[path] = item
	}

	return item
}
