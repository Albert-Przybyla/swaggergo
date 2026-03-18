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
		sourcePath := routerCfg.Source
		if cfg.ProjectRoot != "" && !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(cfg.ProjectRoot, sourcePath)
		}

		parsed, err := ginparser.ParseFile(sourcePath, ginparser.ParseOptions{
			ProjectRoot: cfg.ProjectRoot,
			ModulePath:  cfg.ModulePath,
		})
		if err != nil {
			return fmt.Errorf("failed to parse router file: %w", err)
		}

		spec := buildRouterSpec(cfg, routerCfg, parsed)
		outPath := filepath.Join(cfg.OutputDir, outputFileName(routerCfg.Output))

		if err := generator.WriteYAML(spec, outPath); err != nil {
			return fmt.Errorf("failed to write swagger file: %w", err)
		}
	}

	return nil
}

func outputFileName(name string) string {
	if name == "" {
		return "swagger.yaml"
	}

	return name
}
