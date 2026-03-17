package gin

import "strings"

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
