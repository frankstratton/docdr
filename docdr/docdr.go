package docdr

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func processFieldList(fl *ast.FieldList) string {
	output := ""
	for _, item := range fl.List {
		for _, name := range item.Names {
			output += name.Name
		}
	}
	return output
}

func preFunc(cursor *astutil.Cursor) bool {
	node := cursor.Node()
	fset := token.NewFileSet()

	switch v := node.(type) {
	case *ast.FuncDecl:
		comments := v.Doc

		if comments == nil {
			// Write new comments
			var buf bytes.Buffer
			printer.Fprint(&buf, fset, v)

			s := buf.String()

			fmt.Println(s)
		}
	}
	return true
}

func postFunc(cursor *astutil.Cursor) bool {
	return true
}

func runPackage(pkg *ast.Package) {
	_ = astutil.Apply(pkg, preFunc, postFunc)
}

func ScanPackage(targetDirectory string, targetPackage string) {
	fset := token.NewFileSet()

	pkgs := make(map[string]*ast.Package)
	mode := parser.ParseComments
	var first error

	err := filepath.Walk(targetDirectory,
		func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(info.Name(), ".go") {
				if src, err := parser.ParseFile(fset, path, nil, mode); err == nil {
					name := src.Name.Name
					pkg, found := pkgs[name]
					if !found {
						pkg = &ast.Package{
							Name:  name,
							Files: make(map[string]*ast.File),
						}

						pkgs[name] = pkg
					}

					pkg.Files[path] = src

				} else if first == nil {
					first = err

				}

			}
			return first
		})

	if err != nil {
		log.Fatal(err)
	}

	if targetPackage != "" {
		if p, ok := pkgs[targetPackage]; ok {
			runPackage(p)
		}
	} else {
		for name, _ := range pkgs {
			fmt.Println("\t" + name)
		}

		//for _, p := range pkgs {
		//	runPackage(p)
		//}
	}
}
