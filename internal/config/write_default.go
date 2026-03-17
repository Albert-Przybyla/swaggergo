package config

import (
	"fmt"
	"os"
)

func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file %q already exists", path)
	}

	template := `# swaggen configuration
# Run "swaggen generate" to produce swagger docs

# Your Go module path (from go.mod)
module_path: github.com/mycompany/myapp

# Project root (defaults to current directory)
project_root: .

# Where to write the generated swagger files
output_dir: docs

# Default OpenAPI info (can be overridden per router)
info:
  title: My API
  description: API documentation
  version: 1.0.0
  contact:
    name: API Support
    email: support@example.com

# Default servers
servers:
  - url: http://localhost:8080
    description: Local development
  - url: https://api.example.com
    description: Production

# Router files to analyze. Each entry produces one swagger file.
routers:
  - file: internal/api/router.go
    output: swagger.yaml
    base_path: /api/v1
    info:
      title: My API v1
      version: 1.0.0
    tags:
      users: Users
      products: Products
    security_schemes:
      BearerAuth:
        type: http
        scheme: bearer
        bearer_format: JWT

  # Add more routers for multiple swagger files:
  # - file: internal/admin/router.go
  #   output: admin-swagger.yaml
  #   base_path: /admin
`

	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}

	fmt.Printf("✅ Created %s\n", path)
	fmt.Println("Edit it to match your project, then run: swaggen generate")
	return nil
}
