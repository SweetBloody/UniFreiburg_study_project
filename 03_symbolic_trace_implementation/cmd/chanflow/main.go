package main

import (
	"fmt"
	"log"
	"os"

	loader "github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/01_loader"
	ssa_builder "github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/02_ssa"
	analysis "github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/03_analysis"
	symbolic "github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/04_symbolic"
	report "github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/05_report"
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

	// Build symbolic traces using AST & SSA state
	symBuilder := symbolic.NewBuilder(collector.State)
	symBuilder.Build(prog)

	// Project traces for all channels
	projectedTraces := symBuilder.ProjectAll()

	// Print projected symbolic traces
	report.PrintSymbolicTraces(projectedTraces)
}
