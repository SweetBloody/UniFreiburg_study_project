package main

import (
	"fmt"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

func isChanType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Chan)
	return ok
}

func main() {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  "../example",
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()

		for _, name := range scope.Names() {
			obj := scope.Lookup(name)

			fn, ok := obj.(*types.Func)
			if !ok {
				continue
			}

			sig, ok := fn.Type().(*types.Signature)
			if !ok {
				continue
			}

			params := sig.Params()
			for i := 0; i < params.Len(); i++ {
				p := params.At(i)

				if isChanType(p.Type()) {
					fmt.Printf("Function %s has channel parameter: %s %s\n",
						fn.Name(),
						p.Name(),
						p.Type(),
					)
				}
			}
		}
	}
}
