// Command shortnames fails on any single-letter identifier declared in the given Go files:
// variables, parameters, receivers, fields, functions, types. "_" is fine.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	fileSet := token.NewFileSet()
	found := false
	for _, path := range os.Args[1:] {
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			// A declaration is the identifier its object points back to.
			if ok && len(ident.Name) == 1 && ident.Name != "_" && ident.Obj != nil && ident.Obj.Pos() == ident.Pos() {
				fmt.Printf("%s: single-letter name %q, say what it is\n", fileSet.Position(ident.Pos()), ident.Name)
				found = true
			}
			return true
		})
	}
	if found {
		os.Exit(1)
	}
}
