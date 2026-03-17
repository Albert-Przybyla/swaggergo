package service

import (
	"fmt"
	"path/filepath"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
	"github.com/Albert-Przybyla/swaggergo/internal/generator"
	ginparser "github.com/Albert-Przybyla/swaggergo/internal/parser/gin"
)

func Generate(opts *GenerateOpts) error {
	cfg, err := config.Load(opts.CfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if opts.OutputOverride != "" {
		cfg.OutputDir = opts.OutputOverride
	}

	if opts.Verbose {
		fmt.Printf("Loaded config: %+v\n", cfg)
	}

	for _, routerCfg := range cfg.Routers {
		outFile := routerCfg.Output
		if outFile == "" {
			outFile = "swagger.yaml"
		}

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
		}

		if info.Contact != nil {
			spec.Info.Contact = &generator.ContactObject{
				Name:  info.Contact.Name,
				URL:   info.Contact.URL,
				Email: info.Contact.Email,
			}
		}

		if info.License != nil {
			spec.Info.License = &generator.LicenseObject{
				Name: info.License.Name,
				URL:  info.License.URL,
			}
		}

		for _, s := range cfg.Servers {
			spec.Servers = append(spec.Servers, generator.ServerObject{
				URL:         s.URL,
				Description: s.Description,
			})
		}

		for _, t := range routerCfg.Tags {
			spec.Tags = append(spec.Tags, generator.TagObject{
				Name:        t.Name,
				Description: t.Description,
			})
		}

		spec.Components = &generator.Components{
			Schemas:         map[string]*generator.Schema{},
			SecuritySchemes: map[string]generator.SecurityScheme{},
			Parameters:      map[string]generator.Parameter{},
			Responses:       map[string]generator.Response{},
			RequestBodies:   map[string]generator.RequestBody{},
			Headers:         map[string]generator.Header{},
		}

		if cfg.Components != nil {
			mergeComponents(spec.Components, cfg.Components)
		}

		if routerCfg.Components != nil {
			mergeComponents(spec.Components, routerCfg.Components)
		}

		for name, sec := range routerCfg.SecuritySchemes {
			spec.Components.SecuritySchemes[name] = generator.SecurityScheme{
				Type:         sec.Type,
				Scheme:       sec.Scheme,
				BearerFormat: sec.BearerFormat,
				In:           sec.In,
				Name:         sec.Name,
				Description:  sec.Description,
			}

			spec.Security = append(spec.Security, generator.SecurityRequirement{
				name: {},
			})
		}

		parsed, err := ginparser.ParseFile(routerCfg.Source)
		if err != nil {
			return fmt.Errorf("failed to parse router file: %w", err)
		}

		for _, r := range parsed.Routes {
			fullPath := ginparser.BuildFullPath(
				routerCfg.BasePath,
				r.Group,
				r.Path,
			)

			fullPath = ginparser.NormalizePath(fullPath)

			item := spec.Paths[fullPath]
			if item == nil {
				item = &generator.PathItem{}
				spec.Paths[fullPath] = item
			}

			op := &generator.Operation{
				OperationID: r.Handler,
				Summary:     "",
				Tags:        []string{r.Group},
				Responses: map[string]generator.Response{
					"200": {Description: "OK"},
				},
			}

			switch r.Method {
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

		outPath := filepath.Join(cfg.OutputDir, outFile)

		if err := generator.WriteYAML(spec, outPath); err != nil {
			return fmt.Errorf("failed to write swagger file: %w", err)
		}
	}

	return nil
}

func mergeComponents(dst *generator.Components, src *config.ComponentsConfig) {
	for k, v := range src.Schemas {
		dst.Schemas[k] = convertSchema(v)
	}

	for k, v := range src.SecuritySchemes {
		dst.SecuritySchemes[k] = generator.SecurityScheme{
			Type:         v.Type,
			Scheme:       v.Scheme,
			BearerFormat: v.BearerFormat,
			In:           v.In,
			Name:         v.Name,
			Description:  v.Description,
		}
	}
}

func convertSchema(v interface{}) *generator.Schema {
	return &generator.Schema{
		Type: "object",
	}
}
