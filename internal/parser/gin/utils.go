package gin

import (
	"go/ast"
	"strings"
)

func isHTTPMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return s
}

func trimPointer(s string) string {
	return strings.TrimPrefix(s, "*")
}

func selectorName(sel *ast.SelectorExpr) string {
	var parts []string

	for {
		parts = append([]string{sel.Sel.Name}, parts...)

		switch left := sel.X.(type) {
		case *ast.Ident:
			parts = append([]string{left.Name}, parts...)
			return strings.Join(parts, ".")
		case *ast.SelectorExpr:
			sel = left
		default:
			return strings.Join(parts, ".")
		}
	}
}

func NormalizePath(p string) string {
	parts := strings.Split(p, "/")

	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}

	return strings.Join(parts, "/")
}

func join(parts ...string) string {
	var out []string
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

func BuildFullPath(base, group, route string) string {
	return "/" + join(base, group, route)
}
