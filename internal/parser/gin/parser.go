package gin

import (
	"go/parser"
	"go/token"
)

type ParsedRouter struct {
	Routes []Route
}

func ParseFile(path string) (*ParsedRouter, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	groups := parseGroups(file)
	routes := parseRoutes(file, groups)

	return &ParsedRouter{
		Routes: routes,
	}, nil
}
