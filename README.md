# swaggergo

**swaggergo** swaggergo is a CLI tool for Go that automatically generates OpenAPI/Swagger documentation by analyzing router and handler files. It does not require code annotations — it uses AST (Abstract Syntax Tree) analysis to understand the project structure.

---

## How it works

```
Your Go project
    │
    ├── internal/api/router.go   ← entry point
    │       │
    │       ├── r.GET("/users", handlers.ListUsers)
    │       ├── r.POST("/users", handlers.CreateUser)
    │       └── r.GET("/users/:id", handlers.GetUser)
    │
    └── internal/handlers/
            ├── users.go
            │   ├── func ListUsers(c *gin.Context)
            │   │   ├── c.Query("page")            → query param
            │   │   └── c.JSON(200, UserList{})    → response schema
            │   └── func CreateUser(c *gin.Context)
            │       ├── c.ShouldBindJSON(&req)     → request body
            │       └── c.JSON(201, User{})        → response schema
            └── models/
                └── User{}, CreateUserRequest{}    → JSON schemas
                                │
                                ▼
                    docs/swagger.yaml  ✅
```

## Installation

### Method 1: go install (recommended)

```bash
go install github.com/Albert-Przybyla/swaggergo@latest
```

The binary will be placed in $GOPATH/bin (default: ~/go/bin).
Make sure ~/go/bin is in your $PATH:

```bash
# ~/.bashrc lub ~/.zshrc
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Method 2: from source

```bash
git clone https://github.com/Albert-Przybyla/swaggergo
cd swaggergo
make install        # installs to $GOPATH/bin
# or
make install-global # installs to /usr/local/bin (requires sudo)
```

### Method 3: prebuilt binary (releases)

```bash
# Linux x86_64
curl -L https://github.com/Albert-Przybyla/swaggergo/releases/latest/download/swaggergo-linux-amd64 \
  -o /usr/local/bin/swaggergo && chmod +x /usr/local/bin/swaggergo

# macOS ARM (Apple Silicon)
curl -L https://github.com/Albert-Przybyla/swaggergo/releases/latest/download/swaggergo-darwin-arm64 \
  -o /usr/local/bin/swaggergo && chmod +x /usr/local/bin/swaggergo
```

---

## Quick start

### 1. Initialize configuration in your project

```bash
cd /your/project
swaggergo init
```

This creates a .swaggergo.yaml file with default configuration.

### 2. Edit .swaggergo.yaml

```yaml
# .swaggergo.yaml
module_path: github.com/mycompany/myapp
project_root: .
output_dir: docs

info:
  title: My API
  description: API documentation
  version: 1.0.0
  contact:
    name: API Support
    email: support@example.com

servers:
  - url: http://localhost:8080
    description: Local development
  - url: https://api.example.com
    description: Production

routers:
  - source: internal/admin/api/routes.go
    output: swagger.yaml
    base_path: /api/v1

    info:
      title: My API v1
      version: 1.0.0

    tags:
      - name: users
        description: Users operations
      - name: products
        description: Products operations

    security_schemes:
      BearerAuth:
        type: http
        scheme: bearer
        bearer_format: JWT

    include:
      - /api/v1/*
    exclude:
      - /internal/*
```

### 3. Generate documentation

```bash
swaggergo generate
# or with explicit config:
swaggergo generate --config .swaggergo.yaml --verbose
```

Output: docs/swagger.yaml ready to use in Swagger UI.

---

## Konfiguracja

### Tag mapping

```yaml
tags:
  # key = last segment of handler package path
  users: Users Management
  products: Product Catalog
  orders: Order Processing
  # or full import path:
  github.com/mycompany/myapp/internal/handlers/auth: Authentication
```

---

## Supported frameworks

| Framework             | Status          |
| --------------------- | --------------- |
| **Gin**               | ✅ Full support |
| **net/http** (stdlib) | 🔜 Planned      |
| **Chi**               | 🔜 Planned      |

---

## Swagger UI preview

The easiest way is using Docker:

```bash
docker run -p 8081:8080 \
  -e SWAGGER_JSON=/docs/swagger.yaml \
  -v $(pwd)/docs:/docs \
  swaggerapi/swagger-ui
```

Open: http://localhost:8081

---

## CLI commands

```
swaggergo generate              generate documentation (uses .swaggergo.yaml)
swaggergo generate -c foo.yaml  specify config file
swaggergo generate -o ./out     override output directory
swaggergo generate -v           verbose — shows analysis steps
swaggergo init                  create default .swaggergo.yaml
swaggergo --help                help
```

---

## Tips

**Comments as documentation::**

```go
// ListUsers returns a paginated list of users.
// Supports filtering by role, search by username.
// Use page and page_size query params for pagination.
func ListUsers(c *gin.Context) { ... }
```

→ First line becomes summary, the rest becomes description.
