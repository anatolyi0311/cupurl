package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

type StructInfo struct {
	Name   string
	Target *ast.GenDecl
}

func main() {
	curentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	splitDir := strings.Split(curentDir, string(os.PathSeparator))
	rootDir := filepath.Join(splitDir[:len(splitDir)-2]...)
	dirKey := string(os.PathSeparator) + splitDir[len(splitDir)-3]
	rootDir = string(os.PathSeparator) + rootDir
	// fmt.Println("rootdir:", rootDir)

	givenFiles, err := readFileDir(rootDir, dirKey, "", make(map[string][]string))
	if err != nil {
		log.Fatal(err)
	}
	for filePath, files := range givenFiles {
		// fmt.Println("files:", filePath, "\t", files)
		// создаём token.FileSet
		for _, file := range files {
			fset := token.NewFileSet()
			// текст исходного кода
			// src := readFile(filePath)
			filePath := rootDir[:len(rootDir)-1-len("cupurl")] + filePath + "/" + file
			// получаем дерево разбора
			f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
			if err != nil {
				fmt.Println(err)
			}
			for _, f := range f.Decls {
				genD, ok := f.(*ast.GenDecl)
				if !ok {
					fmt.Printf("SKIP %T is not *ast.GenDecl\n", f)
					continue
				}
				targetStruct := &StructInfo{}
				var thisIsStruct bool
				for _, spec := range genD.Specs {
					currType, ok := spec.(*ast.TypeSpec)
					if !ok {
						fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
						continue
					}

					currStruct, ok := currType.Type.(*ast.StructType)
					if !ok {
						fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
						continue
					}
					targetStruct.Name = currType.Name.Name
					thisIsStruct = true
				}
				//Getting comments
				var needCodegen bool
				var dbeParams string
				if thisIsStruct {
					for _, comment := range genD.Doc.List {
						needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// dbe")
						if len(comment.Text) < 7 {
							dbeParams = ""
						} else {
							dbeParams = strings.Replace(comment.Text, "// dbe:", "", 1)
						}
					}
				}
				fmt.Println("dbeParams", dbeParams)
			}

			// for _, gr := range f.Comments {
			// 	for _, c := range gr.List {
			// 		if strings.HasPrefix(c.Text, "// generate:reset") {
			// 			fmt.Println(fset.Position(c.Slash).String(), c.Text)
			// 			inspect(f, fset)
			// 		}
			// 	}
			// }
		}
	}
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
		// case *ast.TypeSpec:
		// 	// ast.TypeSpec представляет декларацию ...
		// 	fmt.Printf("TypeSpec\nName: %v pos:%v ...: %v %v\n", x.Name.String(), token.Position(x.Pos()), x.Name.Obj.Kind, x.Name.Obj.Name)
		// 	// fmt.Printf("%v ", x.Name.Obj.Kind)
		// 	printer.Fprint(os.Stdout, token, x)
		// 	fmt.Println()
		case *ast.StructType:
			// ast.StructType представляет декларацию struct
			fmt.Printf("StructType\npos:%v fields: %d\n", token.Position(x.Pos()), len(x.Fields.List))
			printer.Fprint(os.Stdout, token, x)
			fmt.Println()
			// case *ast.Comment:
			// 	fmt.Printf("CommentGroup\n%v\n", x.Slash)
			// 	printer.Fprint(os.Stdout, token, x)
			// 	fmt.Println()
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

func readFileDir(rootDir, keyDir, sep string, givenFiles map[string][]string) (map[string][]string, error) {
	/*
		err := filepath.Walk(".",
			func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fmt.Println(path, info.Size())
			return nil
		})
		if err != nil {
			log.Println(err)
		}
	*/
	files, err := os.ReadDir(rootDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			log.Fatal(err)
		}
		if file.Name() == "Makefile" ||
			file.Name() == "profiles" ||
			file.Name() == ".git" ||
			file.Name() == ".github" ||
			file.Name() == ".idea" ||
			strings.HasPrefix(file.Name(), ".git") ||
			strings.Contains(file.Name(), "_mock") ||
			strings.Contains(file.Name(), "_test") ||
			strings.HasSuffix(info.Name(), ".md") ||
			strings.HasSuffix(file.Name(), ".mod") ||
			strings.HasSuffix(file.Name(), ".sum") ||
			strings.HasSuffix(file.Name(), ".conf") ||
			file.Name() == "mocks" {
			continue
		}
		if file.IsDir() {
			// if sep == "" {
			// 	fmt.Println(file.Name())
			// } else {
			// 	fmt.Println(sep, file.Name())
			// }
			subDir := rootDir + string(os.PathSeparator) + file.Name()
			keyDir := keyDir + string(os.PathSeparator) + file.Name()
			readFileDir(subDir, keyDir, sep+"    ", givenFiles)
		}
		if !file.IsDir() {
			// if sep == "" {
			// 	fmt.Println("file: ", file.Name())
			// } else {
			// 	fmt.Println(sep, "file: ", file.Name())
			// }
			givenFiles[keyDir] = append(givenFiles[keyDir], file.Name())
		}
	}
	return givenFiles, nil
}
