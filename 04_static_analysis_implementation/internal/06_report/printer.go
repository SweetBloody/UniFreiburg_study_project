package report

import (
	"fmt"
	"strings"

	matcher "github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/05_matcher"
	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/model"
)

// PrintReport prints the symbolic traces and then prints the deadlock analysis verdict
func PrintReport(projectedTraces map[model.AllocSite]map[model.GoroutineID][]model.TraceNode, deadlocks []matcher.DeadlockReport) {
	PrintSymbolicTraces(projectedTraces)

	if len(deadlocks) == 0 {
		fmt.Println("\nNo deadlocks detected!")
	} else {
		for _, d := range deadlocks {
			fmt.Printf("\nDeadlock detected on channel %s:\n   -> %s\n", d.Channel.Position.String(), d.Reason)
		}
	}
}

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
