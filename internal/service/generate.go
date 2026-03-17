package service

import (
	"fmt"

	"github.com/Albert-Przybyla/swaggergo/internal/config"
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

	return nil
}
