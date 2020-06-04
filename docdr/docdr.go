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
	"sort"
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

func getEditorArgs(editor string, filename string, line int) []string {

	args := []string{}

	if line > 0 {
		args = append(args, fmt.Sprintf("+%d", line))
	}

	switch editor {
	case "gvim":
		args = append(args, "-f") // open in the forground
		args = append(args, filename)
	default:
		args = append(args, filename)
	}

	return args
}

func openFile(filename string, lineNumber int) string {
	runEditor(filename, lineNumber)
	return ""
}

func runEditor(filename string, lineNumber int) {
	editor := os.Getenv("EDITOR")
	path, err := exec.LookPath(editor)
	if err != nil {
		log.Fatal(err)
	}

	args := getEditorArgs(editor, filename, lineNumber)

	cmd := exec.Command(path, args...)
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

	runEditor(tmpFile.Name(), 0)

	defer os.Remove(tmpFile.Name())

	data, err := ioutil.ReadAll(tmpFile)

	if err != nil {
		log.Fatal(err)
	}

	comment := string(data)

	comment = comment[0:strings.Index(comment, fixedMarker)]
	comment = strings.TrimRight(comment, "\n")

	if len(comment) == 0 {
		return ""
	}

	lines := strings.Split(comment, "\n")
	isComment := regexp.MustCompile("^\\s*//")

	for i, line := range lines {
		if !isComment.Match([]byte(line)) {
			lines[i] = "// " + line
		}
	}

	return strings.Join(lines, "\n")

}
func promptForComment(body string, position token.Position) string {
	fmt.Print("[e] to edit, [o] open original file, [s] to skip: ")
	char, _, _ := bufio.NewReader(os.Stdin).ReadRune()

	switch char {
	case 's':
		return ""
	case 'e':
		return openEditor(body)
	case 'o':
		return openFile(position.Filename, position.Line)
	}

	return ""
}

func runPackage(fset *token.FileSet, pkg *ast.Package) {
	for filename, f := range pkg.Files {
		changed := false

		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if ok {
				if fn.Name.IsExported() && fn.Doc.Text() == "" {
					fmt.Println("-----------------------------------------------------")
					fmt.Println(filename)
					fmt.Println("-----------------------------------------------------")
					printNode(fset, fn)
					fmt.Println()

					position := fset.Position(fn.Pos())
					text := promptForComment(nodeString(fset, fn), position)

					fmt.Println(text)

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
					f.Comments = append(f.Comments, cg)
					changed = true
				}
			}
			return true
		})

		if changed {
			writeFile(fset, f, filename)
		}
	}

}

type Stat struct {
	Count    int
	Total    int
	Coverage float64
}

func analyzePackage(fset *token.FileSet, pkg *ast.Package) Stat {
	count := 0
	total := 0

	for _, f := range pkg.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if ok {
				if fn.Name.IsExported() {
					total += 1
					if fn.Doc.Text() != "" {
						count += 1
					}
				}
			}

			return true
		})
	}

	return Stat{
		count,
		total,
		float64(count) / float64(total),
	}
}

func analyzePackages(fset *token.FileSet, packages map[string]*ast.Package) map[string]Stat {
	stats := map[string]Stat{}

	for name, p := range packages {
		stats[name] = analyzePackage(fset, p)
	}

	return stats
}

type StatPair struct {
	Key   string
	Value Stat
}

type StatPairList []StatPair

func (p StatPairList) Len() int           { return len(p) }
func (p StatPairList) Less(i, j int) bool { return p[i].Value.Coverage < p[j].Value.Coverage }
func (p StatPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

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
		stats := analyzePackages(fset, pkgs)

		list := make(StatPairList, len(stats))
		i := 0
		for name, cover := range stats {
			list[i] = StatPair{name, cover}
			i++
		}
		sort.Sort(sort.Reverse(list))

		for _, pairs := range list {
			fmt.Printf("\t%d\t%f\t%s\n", pairs.Value.Total, pairs.Value.Coverage, pairs.Key)
		}

		//for _, p := range pkgs {
		//	runPackage(p, fset)
		//}
	}
}
