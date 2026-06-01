package main

import (
	"fmt"
	"log"
	"os"

	loader "github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/01_loader"
	ssa_builder "github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/02_ssa"
	analysis "github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/03_analysis"
	report "github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/04_report"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: chanflow <package-path>")
		os.Exit(1)
	}
	pattern := os.Args[1]

	// 1. Load packages
	pkgs, err := loader.LoadPackages(pattern)
	if err != nil {
		log.Fatalf("Error loading packages: %v", err)
	}

	// 2. Build SSA
	prog, _ := ssa_builder.BuildSSA(pkgs)

	// 3 & 4. Collect allocation sites and target parameters
	collector := analysis.NewCollector()
	collector.Collect(prog)

	// 5 & 6. Generate constraints
	constraints := analysis.GenerateCallConstraints(prog)

	// 7. Solve constraints
	analysis.Solve(collector.State, constraints)

	// 8. Print result
	report.PrintResults(collector)
}
