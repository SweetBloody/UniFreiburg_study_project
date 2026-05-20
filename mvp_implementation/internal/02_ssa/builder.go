package ssa

import (
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// BuildSSA constructs SSA for all loaded packages
func BuildSSA(pkgs []*packages.Package) (*ssa.Program, []*ssa.Package) {
	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssa.SanityCheckFunctions)
	prog.Build()
	return prog, ssaPkgs
}
