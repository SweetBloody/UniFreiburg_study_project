package main

import (
	"fmt"
	"go/ast"
	"log"

	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  "../example",
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatal(err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		log.Fatal("package loading errors")
	}

	for _, pkg := range pkgs {
		fmt.Println("Package:", pkg.PkgPath)
		fmt.Println("Go files:", len(pkg.Syntax))
		fmt.Println("Type information examples:")

		count := 0

		for expr, tv := range pkg.TypesInfo.Types {
			if count >= 10 {
				break
			}

			pos := pkg.Fset.Position(expr.Pos())

			var text string
			switch e := expr.(type) {
			case *ast.Ident:
				text = e.Name
			case *ast.CallExpr:
				text = "call expression"
			case *ast.ChanType:
				text = "channel type"
			case *ast.UnaryExpr:
				text = "unary expression"
			case *ast.BinaryExpr:
				text = "binary expression"
			default:
				text = fmt.Sprintf("%T", expr)
			}

			fmt.Printf("  %s: %s -> type: %s, value: %v\n",
				pos,
				text,
				tv.Type,
				tv.Value,
			)

			count++
		}
	}
}
