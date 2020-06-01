package docdr

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func nodeString(fset *token.FileSet, node ast.Node) string {
	switch v := node.(type) {
	case *ast.Package:
		for _, f := range v.Files {
			printNode(fset, f)
		}
	default:
	}

	// Write new comments
	var buf bytes.Buffer
	format.Node(&buf, fset, node)

	s := buf.String()

	return s
}
func writeFile(fset *token.FileSet, node ast.Node, filename string) {
	s := nodeString(fset, node)
	ioutil.WriteFile(filename, []byte(s), 0644)
}

func printNode(fset *token.FileSet, node ast.Node) {
	s := nodeString(fset, node)
	fmt.Println(s)
}

func openEditor(body string) string {
	tmpDir := os.TempDir()

	tmpFile, tmpFileErr := ioutil.TempFile(tmpDir, "tempFilePrefix")

	if tmpFileErr != nil {
		fmt.Printf("Error %s while creating tempFile", tmpFileErr)
	}

	// Create reference function with fixedMarker
	fixedMarker := "^^^^ ADD COMMENT ABOVE ---- DO NOT EDIT BELOW THIS LINE ----\n"
	body = fixedMarker + body

	// Write the candidate function into the tempfile first
	err := ioutil.WriteFile(tmpFile.Name(), []byte(body), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// TODO configure based on $EDITOR
	path, err := exec.LookPath("vim")
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(path, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()

	if err != nil {
		fmt.Printf("Start failed: %s", err)
	}

	err = cmd.Wait()

	if err != nil {
		fmt.Printf("Wait failed: %s", err)
	}

	defer os.Remove(tmpFile.Name())

	data, err := ioutil.ReadAll(tmpFile)

	if err != nil {
		log.Fatal(err)
	}

	comment := string(data)

	comment = comment[0:strings.Index(comment, fixedMarker)]
	comment = strings.TrimRight(comment, "\n")

	lines := strings.Split(comment, "\n")
	isComment := regexp.MustCompile("^\\s*//")

	for i, line := range lines {
		if !isComment.Match([]byte(line)) {
			lines[i] = "// " + line
		}
	}

	return strings.Join(lines, "\n")

}
func promptForComment(body string) string {
	fmt.Print("[e] to edit, [s] to skip: ")
	char, _, _ := bufio.NewReader(os.Stdin).ReadRune()

	switch char {
	case 's':
		return ""
	case 'e':
		return openEditor(body)
	}

	return ""
}

func runPackage(fset *token.FileSet, pkg *ast.Package) {
	for filename, f := range pkg.Files {
		comments := []*ast.CommentGroup{}
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CommentGroup)
			if ok {
				comments = append(comments, c)
			}

			fn, ok := n.(*ast.FuncDecl)
			if ok {
				if fn.Name.IsExported() && fn.Doc.Text() == "" {
					fmt.Println("-----------------------------------------------------")
					printNode(fset, fn)
					fmt.Println()

					text := promptForComment(nodeString(fset, fn))

					if text == "" {
						return true
					}

					comment := &ast.Comment{
						Text:  text,
						Slash: fn.Pos() - 1,
					}

					cg := &ast.CommentGroup{
						List: []*ast.Comment{comment},
					}
					fn.Doc = cg
				}
			}
			return true
		})

		// set ast's comments to the collected comments
		f.Comments = comments

		writeFile(fset, f, filename)
	}

}

func ScanPackage(targetDirectory string, targetPackage string) {
	fset := token.NewFileSet()

	pkgs := make(map[string]*ast.Package)

	var first error

	err := filepath.Walk(targetDirectory,
		func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(info.Name(), ".go") {
				if src, err := parser.ParseFile(fset, path, nil, parser.ParseComments); err == nil {
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
			runPackage(fset, p)
		}
	} else {
		for name, _ := range pkgs {
			fmt.Println("\t" + name)
		}

		//for _, p := range pkgs {
		//	runPackage(p, fset)
		//}
	}
}
