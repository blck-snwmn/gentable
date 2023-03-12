package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	source string
	name   string
)

func init() {
	flag.StringVar(&source, "source", "", "")
	flag.StringVar(&name, "name", "", "")
}

func main() {
	flag.Parse()

	if source == "" {
		panic("no source")
	}
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("faild to open(file=%s)\n", source))
	}
	if name == "" {
		panic("no name")
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, source, nil, parser.AllErrors+parser.ParseComments)
	if err != nil {
		panic(err)
	}
	// ast.Print(fset, node)
	tablestr := map[string]bool{}
	newfunc := map[string]bool{}
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.StructType); !ok {
				return true
			}

			structName := n.Name.Name
			if strings.HasSuffix(structName, "Table") {
				return true
			}
			tablestr[structName+"Table"] = false
			newfunc[fmt.Sprintf("New%sTable", structName)] = false
		}
		return true
	})

	astutil.Apply(node, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		switch n := n.(type) {
		case *ast.FuncDecl:
			funcName := n.Name.Name
			if strings.HasPrefix(funcName, "New") && strings.HasSuffix(funcName, "Table") {
				newfunc[funcName] = true
			}
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.StructType); !ok {
				return true
			}
			structName := n.Name.Name
			if strings.HasSuffix(structName, "Table") {
				tablestr[structName] = true
				return true
			}
		}
		return true
	})
	for structName, v := range tablestr {
		if !v {
			node.Decls = append(node.Decls, buildTableStruct(structName))
		}
	}
	for structName, v := range newfunc {
		if !v {
			node.Decls = append(node.Decls, buildNewFunction(structName, name))
		}
	}
	ff, err := os.Create(source)
	if err != nil {
		panic(err)
	}
	astutil.AddImport(fset, node, "github.com/guregu/dynamo")
	// astutil.Imports(fset *token.FileSet, f *ast.File)
	//
	format.Node(ff, fset, node)
}

func buildTableStruct(structName string) *ast.GenDecl {
	st := &ast.TypeSpec{
		Name: &ast.Ident{
			Name: structName,
		},
		Type: &ast.StructType{
			Fields: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "dynamo"},
							Sel: &ast.Ident{Name: "Table"},
						},
					},
				},
			},
		},
	}
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			st,
		},
	}
}

func buildNewFunction(funcName, name string) *ast.FuncDecl {
	structName := strings.TrimPrefix(funcName, "New")
	// funcName := fmt.Sprintf("New%s", structName)
	return &ast.FuncDecl{
		Name: &ast.Ident{
			Name: funcName,
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{{Name: "d"}},
						Type: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "dynamo",
							},
							Sel: &ast.Ident{
								Name: "DB",
							},
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.Ident{
							Name: structName,
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CompositeLit{
							Type: &ast.Ident{
								Name: structName,
							},
							Elts: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X: &ast.Ident{
											Name: "d",
										},
										Sel: &ast.Ident{
											Name: "Table",
										},
									},
									Args: []ast.Expr{
										&ast.BasicLit{
											Kind:  token.STRING,
											Value: name,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
