package main

import (
	"fmt"
	"go/token"
	"log"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
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

	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssa.SanityCheckFunctions)
	prog.Build()

	for _, ssaPkg := range ssaPkgs {
		if ssaPkg == nil {
			continue
		}

		for _, member := range ssaPkg.Members {
			fn, ok := member.(*ssa.Function)
			if !ok || fn.Blocks == nil {
				continue
			}

			fmt.Println("Function:", fn.Name())

			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					pos := prog.Fset.Position(instr.Pos())

					switch v := instr.(type) {
					case *ssa.MakeChan:
						fmt.Printf("  MakeChan at %s: %s\n", pos, v)

					case *ssa.Call:
						fmt.Printf("  Call at %s: %s\n", pos, v)

					case *ssa.Return:
						fmt.Printf("  Return at %s: %s\n", pos, v)

					case *ssa.Phi:
						fmt.Printf("  Phi at %s: %s\n", pos, v)

					default:
						if instr.Pos() != token.NoPos {
							// to discover SSA instructions
							// fmt.Printf("  Instr at %s: %T %s\n", pos, instr, instr)
						}
					}
				}
			}
		}
	}
}
