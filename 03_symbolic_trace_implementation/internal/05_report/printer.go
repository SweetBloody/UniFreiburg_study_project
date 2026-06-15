package report

import (
	"fmt"
	"strings"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/model"
)

// PrintSymbolicTraces iterates over projected channels and prints their traces
func PrintSymbolicTraces(projectedTraces map[model.AllocSite][]model.TraceNode) {
	for site, projected := range projectedTraces {
		fmt.Printf("\nChannel Allocation: %s:%s %s\n", site.Position.Filename, fmt.Sprintf("%d:%d", site.Position.Line, site.Position.Column), site.Type)

		// Format the trace list into a string
		var strs []string
		for _, n := range projected {
			strs = append(strs, n.String())
		}

		if len(strs) > 0 {
			fmt.Printf("Trace: [ %s ]\n", strings.Join(strs, ", "))
		} else {
			fmt.Println("Trace: [ ]")
		}
	}
}
