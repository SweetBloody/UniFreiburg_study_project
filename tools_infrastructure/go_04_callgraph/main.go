package main

import (
	"fmt"
	"log"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
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

	prog, _ := ssautil.AllPackages(pkgs, ssa.SanityCheckFunctions)
	prog.Build()

	cg := static.CallGraph(prog)

	callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		if edge.Caller == nil || edge.Callee == nil {
			return nil
		}

		fmt.Printf("%s -> %s\n", edge.Caller.Func, edge.Callee.Func)
		return nil
	})
}
