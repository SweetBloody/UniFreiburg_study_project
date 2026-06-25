package report

import (
	"fmt"
	"strings"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/model"
)

// PrintSymbolicTraces iterates over projected channels and prints their traces
func PrintSymbolicTraces(projectedTraces map[model.AllocSite]map[model.GoroutineID][]model.TraceNode) {
	for site, siteTraces := range projectedTraces {
		fmt.Printf("\nChannel Allocation: %s:%s %s\n", site.Position.Filename, fmt.Sprintf("%d:%d", site.Position.Line, site.Position.Column), site.Type)

		for gID, trace := range siteTraces {
			var strs []string
			for _, n := range trace {
				strs = append(strs, n.String())
			}

			if len(strs) > 0 {
				fmt.Printf("  Goroutine %s: [ %s ]\n", gID, strings.Join(strs, ", "))
			} else {
				fmt.Printf("  Goroutine %s: [ ]\n", gID)
			}
		}
	}
}
