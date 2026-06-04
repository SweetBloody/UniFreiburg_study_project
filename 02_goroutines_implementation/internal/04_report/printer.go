package report

import (
	"fmt"
	"sort"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/02_goroutines_implementation/internal/model"

	analysis "github.com/SweetBloody/UniFreiburg_study_project/chanflow/02_goroutines_implementation/internal/03_analysis"
)

// PrintResults prints the abstract domain: which goroutines perform what operations on each channel
func PrintResults(collector *analysis.Collector) {
	domain := make(map[model.AllocSite]map[model.GoroutineID]map[model.OpType]struct{})

	for _, op := range collector.Operations {
		sites := collector.State[op.ChannelVar]
		for site := range sites {
			if domain[site] == nil {
				domain[site] = make(map[model.GoroutineID]map[model.OpType]struct{})
			}
			if domain[site][op.ChannelVar.Goroutine] == nil {
				domain[site][op.ChannelVar.Goroutine] = make(map[model.OpType]struct{})
			}
			domain[site][op.ChannelVar.Goroutine][op.Type] = struct{}{}
		}
	}

	var sites []model.AllocSite
	for site := range domain {
		sites = append(sites, site)
	}
	sort.Slice(sites, func(i, j int) bool {
		return sites[i].Position.String() < sites[j].Position.String()
	})

	for _, site := range sites {
		fmt.Printf("Channel Allocation: %s %s\n", site.Position, site.Type)

		gMap := domain[site]
		var gIDs []model.GoroutineID
		for gID := range gMap {
			gIDs = append(gIDs, gID)
		}
		sort.Slice(gIDs, func(i, j int) bool {
			return string(gIDs[i]) < string(gIDs[j])
		})

		for _, gID := range gIDs {
			fmt.Printf("- Goroutine '%s':\n", string(gID))

			opMap := gMap[gID]
			var ops []string
			for op := range opMap {
				ops = append(ops, string(op))
			}
			sort.Strings(ops)

			for _, op := range ops {
				fmt.Printf("    - %s\n", op)
			}
		}
		fmt.Println()
	}

	if len(sites) == 0 {
		fmt.Println("No channel usages found.")
	}
}
