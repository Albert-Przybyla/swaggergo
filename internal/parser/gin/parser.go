package gin

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ParsedRouter struct {
	Routes  []Route
	Schemas map[string]*Schema
}

type ParseOptions struct {
	ProjectRoot string
	ModulePath  string
}

type packageContext struct {
	routerFile     *ast.File
	routerFileInfo *fileInfo
	routerPkg      *packageInfo
	files          []*ast.File
	funcs          map[string][]*functionInfo
	structs        map[string]*structInfo
	packages       map[string]*packageInfo
}

type functionInfo struct {
	decl *ast.FuncDecl
	file *fileInfo
}

type structInfo struct {
	key        string
	name       string
	typeParams []string
	file       *fileInfo
	structType *ast.StructType
	doc        *ast.CommentGroup
}

type packageInfo struct {
	name       string
	importPath string
	dir        string
}

type fileInfo struct {
	path    string
	file    *ast.File
	pkg     *packageInfo
	imports map[string]string
}

func ParseFile(path string, opts ParseOptions) (*ParsedRouter, error) {
	ctx, err := parsePackage(path, opts)
	if err != nil {
		return nil, err
	}

	groups := parseGroups(ctx.routerFile)
	routes := parseRoutes(ctx, groups)

	return &ParsedRouter{
		Routes:  routes,
		Schemas: collectRouteSchemas(routes, ctx),
	}, nil
}

func parsePackage(path string, opts ParseOptions) (*packageContext, error) {
	fset := token.NewFileSet()
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse package: %w", err)
	}

	for _, pkg := range pkgs {
		ctx := &packageContext{
			funcs:    make(map[string][]*functionInfo),
			structs:  make(map[string]*structInfo),
			packages: make(map[string]*packageInfo),
		}

		routerPkg := &packageInfo{
			name:       pkg.Name,
			importPath: buildImportPath(opts, dir),
			dir:        dir,
		}
		ctx.routerPkg = routerPkg
		if routerPkg.importPath != "" {
			ctx.packages[routerPkg.importPath] = routerPkg
		}

		for filePath, file := range pkg.Files {
			fileInfo := newFileInfo(filePath, file, routerPkg)

			if filepath.Base(filePath) == base {
				ctx.routerFile = file
				ctx.routerFileInfo = fileInfo
			}

			ctx.files = append(ctx.files, file)
			indexPackageDecls(ctx, fileInfo)
		}

		if ctx.routerFile == nil {
			continue
		}

		if err := indexProjectStructs(ctx, opts, fset); err != nil {
			return nil, err
		}

		return ctx, nil
	}

	return nil, fmt.Errorf("router file %s not found in parsed package", path)
}

func newFileInfo(path string, file *ast.File, pkg *packageInfo) *fileInfo {
	info := &fileInfo{
		path:    path,
		file:    file,
		pkg:     pkg,
		imports: make(map[string]string),
	}

	for _, spec := range file.Imports {
		importPath := trimQuotes(spec.Path.Value)
		alias := filepath.Base(importPath)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		info.imports[alias] = importPath
	}

	return info
}

func indexPackageDecls(ctx *packageContext, fileInfo *fileInfo) {
	for _, decl := range fileInfo.file.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			ctx.funcs[typed.Name.Name] = append(ctx.funcs[typed.Name.Name], &functionInfo{
				decl: typed,
				file: fileInfo,
			})
		case *ast.GenDecl:
			indexStructDecls(ctx, fileInfo, typed, true)
		}
	}
}

func indexProjectStructs(ctx *packageContext, opts ParseOptions, fset *token.FileSet) error {
	if opts.ProjectRoot == "" {
		return nil
	}

	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}

	return filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || strings.HasPrefix(name, ".") {
				if path == root {
					return nil
				}
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		dir := filepath.Dir(path)
		importPath := buildImportPath(opts, dir)
		pkgInfo := ctx.packages[importPath]
		if pkgInfo == nil {
			pkgInfo = &packageInfo{
				name:       file.Name.Name,
				importPath: importPath,
				dir:        dir,
			}
			if importPath != "" {
				ctx.packages[importPath] = pkgInfo
			}
		}

		fileInfo := newFileInfo(path, file, pkgInfo)
		indexPackageDecls(ctx, fileInfo)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			indexStructDecls(ctx, fileInfo, gen, false)
		}

		return nil
	})
}

func indexStructDecls(ctx *packageContext, fileInfo *fileInfo, gen *ast.GenDecl, localPackage bool) {
	if gen == nil || gen.Tok != token.TYPE {
		return
	}

	for _, spec := range gen.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		key := typeKeyForFile(fileInfo, typeSpec.Name.Name, localPackage)
		doc := typeSpec.Doc
		if doc == nil {
			doc = gen.Doc
		}

		ctx.structs[key] = &structInfo{
			key:        key,
			name:       typeSpec.Name.Name,
			typeParams: extractTypeParams(typeSpec),
			file:       fileInfo,
			structType: structType,
			doc:        doc,
		}
	}
}

func extractTypeParams(typeSpec *ast.TypeSpec) []string {
	if typeSpec == nil || typeSpec.TypeParams == nil {
		return nil
	}

	var out []string
	for _, field := range typeSpec.TypeParams.List {
		for _, name := range field.Names {
			out = append(out, name.Name)
		}
	}

	return out
}

func typeKeyForFile(fileInfo *fileInfo, typeName string, localPackage bool) string {
	if localPackage || fileInfo == nil || fileInfo.pkg == nil {
		return typeName
	}

	return fileInfo.pkg.name + "." + typeName
}

func buildImportPath(opts ParseOptions, dir string) string {
	if opts.ProjectRoot == "" || opts.ModulePath == "" {
		return ""
	}

	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return ""
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}

	rel, err := filepath.Rel(root, absDir)
	if err != nil || rel == "." {
		return opts.ModulePath
	}

	return strings.TrimSuffix(opts.ModulePath, "/") + "/" + filepath.ToSlash(rel)
}
