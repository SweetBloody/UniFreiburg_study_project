package main

import (
	"fmt"
	"log"
	"os"

	loader "02_goroutines_implementation/internal/01_loader"
	ssa_builder "02_goroutines_implementation/internal/02_ssa"
	analysis "02_goroutines_implementation/internal/03_analysis"
	report "02_goroutines_implementation/internal/04_report"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: chanflow <package-path>")
		os.Exit(1)
	}
	pattern := os.Args[1]

	// Load packages
	pkgs, err := loader.LoadPackages(pattern)
	if err != nil {
		log.Fatalf("Error loading packages, error: %v", err)
	}

	// Build SSA
	prog, _ := ssa_builder.BuildSSA(pkgs)

	// Traverse call graph, collect MakeChan, generate constraints, and collect Operations
	collector := analysis.NewCollector()
	collector.Collect(prog)

	// Solve constraints
	analysis.Solve(collector.State, collector.Constraints)

	// Print result
	report.PrintResults(collector)
}
