package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
)

func main() {

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("dir:", dir)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("homeDir:", homeDir)

	// absolute path to file for inspect.
	root := homeDir + "/yapracticum/cupurl/"
	path := []string{
		"cmd/shortener/main.go",
		"internal/app/auth/auth.go",
		"internal/app/handlers/handlers.go",
	}
	filePath := root + path[len(path)-1]

	// создаём token.FileSet
	fset := token.NewFileSet()
	// текст исходного кода
	src := readFile(filePath)
	// получаем дерево разбора
	f, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		fmt.Println(err)
	}
	inspect(f, fset)
}

func inspect(node ast.Node, token *token.FileSet) {
	fmt.Println()
	// запускаем инспектор, который рекурсивно обходит ветви AST
	// передаём инспектирующую функцию анонимно
	ast.Inspect(node, func(n ast.Node) bool {
		// проверяем, какой конкретный тип лежит в узле
		switch x := n.(type) {
		// case *ast.CallExpr:
		// 	// ast.CallExpr представляет вызов функции или метода
		// 	fmt.Printf("CallExpr %v: ", token.Position(x.Fun.Pos()))
		// 	printer.Fprint(os.Stdout, token, x)
		// 	fmt.Println()
		// case *ast.FuncDecl:
		// 	// ast.FuncDecl представляет декларацию функции
		// 	fmt.Printf("FuncDecl %s %v: ", x.Name.Name, token.Position(x.Pos()))
		// 	printer.Fprint(os.Stdout, token, x)
		// 	fmt.Println()
		case *ast.TypeSpec:
			// ast.TypeSpec представляет декларацию ...
			fmt.Printf("TypeSpec\nName: %v pos:%v ...: %v %v\n", x.Name.String(), token.Position(x.Pos()), x.Name.Obj.Kind, x.Name.Obj.Name)
			// fmt.Printf("%v ", x.Name.Obj.Kind)
			printer.Fprint(os.Stdout, token, x)
			fmt.Println()
		case *ast.StructType:
			// ast.StructType представляет декларацию struct
			fmt.Printf("StructType\nStruct: %v pos:%v fields: %d\n%v\n", x.Struct, token.Position(x.Pos()), len(x.Fields.List), x)
			printer.Fprint(os.Stdout, token, x)
			fmt.Println()
		case *ast.Comment:
			fmt.Printf("CommentGroup\n%v\n", x.Slash)
			printer.Fprint(os.Stdout, token, x)
			fmt.Println()
		}
		return true
	})
}

func readFile(filePath string) string {
	// Read the entire file into a byte slice
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	// Print the content (convert byte slice to string)
	// fmt.Println(string(content))
	return string(content)
}
