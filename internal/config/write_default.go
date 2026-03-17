package config

import (
	"embed"
	"fmt"
	"os"
)

//go:embed default_template.yaml
var templateFS embed.FS

func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file %q already exists", path)
	}

	data, err := templateFS.ReadFile("default_template.yaml")
	if err != nil {
		return fmt.Errorf("cannot read embedded template: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}

	fmt.Printf("Created %s\n", path)
	fmt.Println("Edit it to match your project, then run: swaggen generate")

	return nil
}
