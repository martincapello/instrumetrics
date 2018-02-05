package main

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// Block represents the information about a basic block to be recorded in the analysis.
// Note: Our definition of basic block is based on control structures; we don't break
// apart && and ||. We could but it doesn't seem important enough to bother.
type Block struct {
	startByte token.Pos
	endByte   token.Pos
	numStmt   int
}

// File is a wrapper for the state of a file used in the parser.
// The basic parse tree walker is a method of this type.
type File struct {
	fset      *token.FileSet
	name      string // Name of file.
	astFile   *ast.File
	blocks    []Block
	atomicPkg string // Package name for "sync/atomic" in this file.
}

// Visit implements the ast.Visitor interface.
func (f *File) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		//start := f.fset.Position(n.Pos())
		//end := f.fset.Position(n.End())
		// Prints information about functions
		//fmt.Printf("name: %v start: %v:%v end: %v:%v\n", n.Name, start.Line, start.Column, end.Line, end.Column)

		newList := []ast.Stmt{f.newCheckpoint("start")}
		newList = append(newList, n.Body.List...)
		n.Body.List = append(newList, f.newCheckpoint("stop"))

	}
	return f
}

func (f *File) newCheckpoint(kind string) ast.Stmt {
	s := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.Ident{
				Name: kind,
			},
			Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"" + kind + " measuring something...\""}},
		},
	}
	return s
}

func main() {
	// run.go_ is a copy of GOPATH$/src/github.com/spiffe/spire/cmd/spire-agent/cli/run/run.go
	// to use as an example of how instrumentation could work.
	// Basicly the code herein was copied from the cover golang tool and then adapted.
	name := "run.go_"
	fset := token.NewFileSet()
	content, err := ioutil.ReadFile(name)
	if err != nil {
		log.Fatalf("cover: %s: %s", name, err)
	}
	parsedFile, err := parser.ParseFile(fset, name, content, parser.ParseComments)
	if err != nil {
		log.Fatalf("cover: %s: %s", name, err)
	}
	parsedFile.Comments = trimComments(parsedFile, fset)

	file := &File{
		fset:    fset,
		name:    name,
		astFile: parsedFile,
	}

	ast.Walk(file, file.astFile)
	fd := os.Stdout

	file.print(fd)
}

func (f *File) print(w io.Writer) {
	printer.Fprint(w, f.fset, f.astFile)
}

// trimComments drops all but the //go: comments, some of which are semantically important.
// We drop all others because they can appear in places that cause our counters
// to appear in syntactically incorrect places. //go: appears at the beginning of
// the line and is syntactically safe.
func trimComments(file *ast.File, fset *token.FileSet) []*ast.CommentGroup {
	var comments []*ast.CommentGroup
	for _, group := range file.Comments {
		var list []*ast.Comment
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") && fset.Position(comment.Slash).Column == 1 {
				list = append(list, comment)
			}
		}
		if list != nil {
			comments = append(comments, &ast.CommentGroup{List: list})
		}
	}
	return comments
}
