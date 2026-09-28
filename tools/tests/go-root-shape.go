//go:build ignore

// Command go-root-shape performs dependency-free structural checks that catch
// package-level source regressions even when the pinned toolchain/modules are
// unavailable on a packaging host. It is not a substitute for go build/test.
package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type where struct {
	file string
	line int
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go-root-shape <repository-root>")
		os.Exit(2)
	}
	dir := os.Args[1]
	fset := token.NewFileSet()
	decls := map[string][]where{}
	var badWriteJSON []where
	var namedResultRedeclare []where
	parseFailed := false
	files := 0

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		match, err := build.Default.MatchFile(dir, entry.Name())
		if err != nil {
			fmt.Printf("MATCH %s: %v\n", entry.Name(), err)
			parseFailed = true
			continue
		}
		if !match {
			continue
		}
		files++
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			fmt.Printf("PARSE %s: %v\n", entry.Name(), err)
			parseFailed = true
			continue
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					pos := fset.Position(d.Pos())
					key := "func " + d.Name.Name
					decls[key] = append(decls[key], where{entry.Name(), pos.Line})
				}
				checkNamedResultRedeclare(fset, entry.Name(), d, &namedResultRedeclare)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						pos := fset.Position(ts.Pos())
						key := "type " + ts.Name.Name
						decls[key] = append(decls[key], where{entry.Name(), pos.Line})
					}
				}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if ok && id.Name == "writeJSON" && len(call.Args) != 2 {
				pos := fset.Position(call.Pos())
				badWriteJSON = append(badWriteJSON, where{entry.Name(), pos.Line})
			}
			return true
		})
	}

	var duplicateKeys []string
	for key, locations := range decls {
		if len(locations) > 1 {
			duplicateKeys = append(duplicateKeys, key)
		}
	}
	sort.Strings(duplicateKeys)
	for _, key := range duplicateKeys {
		fmt.Printf("DUP %s:", key)
		for _, loc := range decls[key] {
			fmt.Printf(" %s:%d", loc.file, loc.line)
		}
		fmt.Println()
	}
	for _, loc := range badWriteJSON {
		fmt.Printf("BAD_WRITEJSON %s:%d\n", loc.file, loc.line)
	}
	for _, loc := range namedResultRedeclare {
		fmt.Printf("NAMED_RESULT_REDECLARE %s:%d\n", loc.file, loc.line)
	}
	if parseFailed || len(duplicateKeys) > 0 || len(badWriteJSON) > 0 || len(namedResultRedeclare) > 0 {
		os.Exit(1)
	}
	fmt.Printf("ROOT_GO_SHAPE_PASS files=%d\n", files)
}

func checkNamedResultRedeclare(fset *token.FileSet, fileName string, fn *ast.FuncDecl, out *[]where) {
	if fn.Type.Results == nil || fn.Body == nil {
		return
	}
	results := map[string]struct{}{}
	for _, field := range fn.Type.Results.List {
		for _, name := range field.Names {
			results[name.Name] = struct{}{}
		}
	}
	if len(results) == 0 {
		return
	}
	for _, stmt := range fn.Body.List {
		declStmt, ok := stmt.(*ast.DeclStmt)
		if !ok {
			continue
		}
		gen, ok := declStmt.Decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			values, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range values.Names {
				if _, conflict := results[name.Name]; conflict {
					pos := fset.Position(name.Pos())
					*out = append(*out, where{fileName, pos.Line})
				}
			}
		}
	}
}
