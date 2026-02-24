package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

const typStr = "string"

type StructInfo struct {
	Name   string
	Target *ast.GenDecl
}

type DataCode struct {
	File    string
	Package string
	Structs []templateData
}

type FieldsStruct struct {
	Symb string
	Name  string
	IsInt bool
	IsStr bool
	IsArr bool
	IsMap bool
}

type templateData struct {
	Name   string
	Symb   string
	Fields []FieldsStruct
	Child  bool
	path   string
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

	givenFiles, err := readFileDir(rootDir, dirKey, "", make(map[string][]string))
	if err != nil {
		log.Fatal(err)
	}
	for filePath, files := range givenFiles {
		// создаём token.FileSet
		for _, file := range files {
			pkg := strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1] // file[:len(file)-len(".go")]
			path := rootDir[:len(rootDir)-1-len("cupurl")] + filePath
			fpath := path + "/" + file
			getStructAndComments(fpath, pkg, path)
		}
	}
}

func getStructAndComments(fpath, pkg, path string) {
	// var sb strings.Builder
	// fmt.Fprintf(&sb, "%s", methodResetTmpl)
	// текст исходного кода
	// src := readFile(filePath)
	// fpath := rootDir[:len(rootDir)-1-len("cupurl")] + filePath + "/" + file

	var methodResetTmpl = `//CODE GENERATED AUTOMATICALLY
package {{.Package}}

{{range .Structs}}
func ({{.Symb}} *{{.Name}}) Reset() {
    if {{.Symb}} == nil {
        return
    }{{if .Child}}{{range .Fields}}
    if resetter, ok := {{.Name}}.(interface{ Reset() }); ok && {{.Name}} != nil {
        resetter.Reset()
    }{{end}}{{end}}
}{{end}}
`
	var data DataCode
	data.Structs = make([]templateData, 0)

	fset := token.NewFileSet()

	// получаем дерево разбора
	f, err := parser.ParseFile(fset, fpath, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
	}
	for _, f := range f.Decls {
		genD, ok := f.(*ast.GenDecl)
		if !ok {
			continue
		}
		structs := make(map[string]*ast.StructType)

		for _, spec := range genD.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			currStruct, ok := currType.Type.(*ast.StructType)
			if !ok {
				continue
			}
			structs[currType.Name.Name] = currStruct
		}

		if len(structs) > 0 {
			var needCodegen bool
			//Getting comments
			for _, comment := range genD.Doc.List {
				prefix := "// generate:reset"
				needCodegen = strings.HasPrefix(comment.Text, prefix)
				// ...
				if needCodegen {
					var fieldsStruct []FieldsStruct
					data.File = pkg + ".go"
					data.Package = pkg
					// ...
					for k, v := range structs {
						// fmt.Println(comment.Text)
						// fmt.Println("type:", k)
						symb := strings.ToLower(k[0:1])
						for _, field := range v.Fields.List {
							if len(field.Names) < 1 {
								continue
							}
							fieldName := symb + "." + field.Names[0].String()
							fieldStruct := FieldsStruct{Name: fieldName}
							// if _, ok := field.Type.(*ast.StructType); ok {
							// }
							if _, ok := field.Type.(*ast.ChanType); ok {
								continue
							}
							if _, ok := field.Type.(*ast.Ident); ok {
								fmt.Println("    ..field.Ident", fieldName, field.Type)
								continue
							}
							if _, ok := field.Type.(*ast.ArrayType); ok {
								fmt.Println("    ..field.Array", fieldName, field.Type)
								fieldStruct.IsArr = true
							}
							if _, ok := field.Type.(*ast.InterfaceType); ok {
								fmt.Println("  --field", fieldName, field.Type)
							}
							fieldsStruct = append(fieldsStruct, fieldStruct)
							fmt.Println("  field", fieldName, field.Type)
						}
						// ...
						tmpl := templateData{
							Name:   k,
							Symb:   symb,
							Fields: fieldsStruct,
							Child:  len(fieldsStruct) > 0,
						}
						data.Structs = append(data.Structs, tmpl)
					}
				}
			}
		}
	}
	// ...
	if strings.Split(fpath, "/")[len(strings.Split(fpath, "/"))-1] == data.File {
		var buf bytes.Buffer
		t := template.Must(template.New("struct-method").Parse(methodResetTmpl))
		if err := t.Execute(&buf, data); err != nil {
			fmt.Println(err)
		}
		bufFmt, err := format.Source(buf.Bytes())
		if err != nil {
			panic(err)
		}
		// basename := strings.TrimSuffix(file, filepath.Ext(file))
		err = os.WriteFile(path+"/"+"reset.gen.go", bufFmt, 0644)
	}
}

// func readFile(filePath string) string {
// 	// Read the entire file into a byte slice
// 	content, err := os.ReadFile(filePath)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// Print the content (convert byte slice to string)
// 	// fmt.Println(string(content))
// 	return string(content)
// }

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
			file.Name() == "reset" ||
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
