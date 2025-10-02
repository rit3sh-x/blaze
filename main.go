package main

import (
	"fmt"

	"github.com/rit3sh-x/blaze/core/ast"
)

func main() {
	tree, err := ast.BuildSchemaAST("", "")
	if err != nil {
		fmt.Println("error")
	}
	fmt.Println(tree)
	fmt.Println("Success")
}