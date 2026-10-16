package codelint

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// unit is one package ready for analysis: parsed files plus whatever types are known.
type unit struct {
	pkg     *types.Package
	files   []*ast.File
	info    *types.Info
	sizes   types.Sizes
	module  string
	imports []string
}

// loadTyped loads every package under root with full type information. It fails if any
// package has a load or type error, so the caller can fall back.
func loadTyped(root string, load Load) ([]*unit, *token.FileSet, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Dir:        root,
		Fset:       fset,
		Env:        load.Env,
		BuildFlags: load.BuildFlags,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, nil, err
	}
	var units []*unit
	var skipped error
	for _, p := range pkgs {
		if len(p.CompiledGoFiles) == 0 {
			// e.g. every file excluded by build constraints, or no module at all
			if len(p.Errors) > 0 && skipped == nil {
				skipped = p.Errors[0]
			}
			continue
		}
		if len(p.Errors) > 0 {
			return nil, nil, p.Errors[0]
		}
		u := &unit{pkg: p.Types, files: p.Syntax, info: p.TypesInfo, sizes: p.TypesSizes}
		if p.Module != nil {
			u.module = p.Module.Path
		}
		for imp := range p.Imports {
			u.imports = append(u.imports, imp)
		}
		units = append(units, u)
	}
	if len(units) == 0 && hasGoFiles(root) {
		if skipped != nil {
			return nil, nil, skipped
		}
		return nil, nil, errors.New("no packages matched ./...")
	}
	return units, fset, nil
}

// loadSyntax parses the non-test Go files under root, one package per directory, and
// type-checks them against each other only. Imports from outside root become empty
// packages, so qualified names still resolve to their import path.
func loadSyntax(root string) ([]*unit, *token.FileSet, error) {
	fset := token.NewFileSet()
	modDir, modPath := findModule(root)
	byDir := map[string][]*ast.File{}
	var dirs []string
	ctxt := build.Default
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if ok, _ := ctxt.MatchFile(filepath.Dir(p), name); !ok {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments|parser.SkipObjectResolution)
		if f == nil {
			return err
		}
		dir := filepath.Dir(p)
		if byDir[dir] == nil {
			dirs = append(dirs, dir)
		}
		byDir[dir] = append(byDir[dir], f)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(dirs)
	pkgPath := func(dir string) string {
		base, prefix := root, "command-line-arguments"
		if modDir != "" {
			base, prefix = modDir, modPath
		}
		rel, _ := filepath.Rel(base, dir)
		if rel == "." {
			return prefix
		}
		return path.Join(prefix, filepath.ToSlash(rel))
	}
	units := map[string]*unit{}
	var order []string
	for _, dir := range dirs {
		files := byDir[dir]
		// a directory holds one package: keep the files that agree with the first
		name := files[0].Name.Name
		var keep []*ast.File
		for _, f := range files {
			if f.Name.Name == name {
				keep = append(keep, f)
			}
		}
		p := pkgPath(dir)
		u := &unit{files: keep, module: modPath, sizes: types.SizesFor("gc", runtime.GOARCH)}
		seen := map[string]bool{}
		for _, f := range keep {
			for _, s := range f.Imports {
				ip := strings.Trim(s.Path.Value, "`\"")
				if !seen[ip] {
					seen[ip] = true
					u.imports = append(u.imports, ip)
				}
			}
		}
		u.pkg = types.NewPackage(p, name)
		units[p] = u
		order = append(order, p)
	}
	var out []*unit
	for _, u := range topo(units, order) {
		u.info = newInfo()
		conf := types.Config{
			Importer: importerFunc(func(ip string) (*types.Package, error) {
				if dep, ok := units[ip]; ok && dep.pkg.Complete() {
					return dep.pkg, nil
				}
				pkg := types.NewPackage(ip, guessName(ip))
				pkg.MarkComplete()
				return pkg, nil
			}),
			Error: func(error) {},
			Sizes: u.sizes,
		}
		pkg, _ := conf.Check(u.pkg.Path(), fset, u.files, u.info)
		pkg.MarkComplete()
		u.pkg = pkg
		out = append(out, u)
	}
	return out, fset, nil
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(p string) (*types.Package, error) { return f(p) }

func newInfo() *types.Info {
	return &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Implicits:  map[ast.Node]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
		Instances:  map[*ast.Ident]types.Instance{},
	}
}

// guessName is the package name an import path most likely declares.
func guessName(ip string) string {
	elems := strings.Split(ip, "/")
	name := elems[len(elems)-1]
	if len(elems) > 1 && len(name) > 1 && name[0] == 'v' && strings.Trim(name[1:], "0123456789") == "" {
		name = elems[len(elems)-2]
	}
	name = strings.TrimPrefix(name, "go-")
	name = strings.TrimSuffix(name, ".go")
	name = strings.TrimSuffix(name, "-go")
	return strings.NewReplacer("-", "", ".", "").Replace(name)
}

// findModule returns the directory and path of the go.mod at or above dir, if any.
func findModule(dir string) (string, string) {
	for d := dir; ; d = filepath.Dir(d) {
		if b, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			for _, ln := range strings.Split(string(b), "\n") {
				if f := strings.Fields(ln); len(f) == 2 && f[0] == "module" {
					return d, strings.Trim(f[1], "\"`")
				}
			}
			return d, ""
		}
		if filepath.Dir(d) == d {
			return "", ""
		}
	}
}

func hasGoFiles(root string) bool {
	found := false
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// topo orders units so that each comes after the units it imports.
func topo(units map[string]*unit, order []string) []*unit {
	var out []*unit
	state := map[string]int{}
	var visit func(p string)
	visit = func(p string) {
		if state[p] != 0 {
			return
		}
		state[p] = 1
		u := units[p]
		imps := append([]string(nil), u.imports...)
		sort.Strings(imps)
		for _, ip := range imps {
			if _, ok := units[ip]; ok {
				visit(ip)
			}
		}
		state[p] = 2
		out = append(out, u)
	}
	for _, p := range order {
		visit(p)
	}
	return out
}

// run applies the analyzers (and their prerequisites) to the units in dependency order,
// carrying package facts between units in memory.
func run(analyzers []*analysis.Analyzer, units []*unit, fset *token.FileSet) ([]analysis.Diagnostic, error) {
	byPath := map[string]*unit{}
	var order []string
	for _, u := range units {
		byPath[u.pkg.Path()] = u
		order = append(order, u.pkg.Path())
	}
	sort.Strings(order)
	var all []*analysis.Analyzer
	seen := map[*analysis.Analyzer]bool{}
	var add func(a *analysis.Analyzer)
	add = func(a *analysis.Analyzer) {
		if seen[a] {
			return
		}
		seen[a] = true
		for _, r := range a.Requires {
			add(r)
		}
		all = append(all, a)
	}
	for _, a := range analyzers {
		add(a)
	}
	type factKey struct {
		pkg string
		typ reflect.Type
	}
	facts := map[*analysis.Analyzer]map[factKey]analysis.Fact{}
	var diags []analysis.Diagnostic
	for _, u := range topo(byPath, order) {
		results := map[*analysis.Analyzer]any{}
		for _, a := range all {
			if facts[a] == nil {
				facts[a] = map[factKey]analysis.Fact{}
			}
			store := facts[a]
			mod := &analysis.Module{Path: u.module}
			pass := &analysis.Pass{
				Analyzer:   a,
				Fset:       fset,
				Files:      u.files,
				Pkg:        u.pkg,
				TypesInfo:  u.info,
				TypesSizes: u.sizes,
				Module:     mod,
				ResultOf:   map[*analysis.Analyzer]any{},
				ReadFile:   os.ReadFile,
				Report: func(d analysis.Diagnostic) {
					if d.Category == "" {
						d.Category = a.Name
					}
					diags = append(diags, d)
				},
				ImportObjectFact: func(types.Object, analysis.Fact) bool { return false },
				ExportObjectFact: func(types.Object, analysis.Fact) {},
				ImportPackageFact: func(p *types.Package, f analysis.Fact) bool {
					got, ok := store[factKey{p.Path(), reflect.TypeOf(f)}]
					if ok {
						reflect.ValueOf(f).Elem().Set(reflect.ValueOf(got).Elem())
					}
					return ok
				},
				ExportPackageFact: func(f analysis.Fact) {
					store[factKey{u.pkg.Path(), reflect.TypeOf(f)}] = f
				},
				AllObjectFacts:  func() []analysis.ObjectFact { return nil },
				AllPackageFacts: func() []analysis.PackageFact { return nil },
			}
			for _, r := range a.Requires {
				pass.ResultOf[r] = results[r]
			}
			res, err := a.Run(pass)
			if err != nil {
				return nil, fmt.Errorf("%s on %s: %w", a.Name, u.pkg.Path(), err)
			}
			results[a] = res
		}
	}
	return diags, nil
}
