package report

import (
	"fmt"
	"sort"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/model"

	analysis "github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/03_analysis"
)

// PrintResults prints the possible channel allocation sites for each target parameter
func PrintResults(collector *analysis.Collector) {
	for _, target := range collector.TargetParameters {
		param := target.Param
		id := target.ID

		fmt.Printf("Function parameter: %s %s\n\n", id, param.Type())

		sitesMap := collector.State[id]
		if len(sitesMap) == 0 {
			fmt.Println("Possible channel allocation sites: None")
		} else {
			fmt.Println("Possible channel allocation sites:")

			// Sort sites for deterministic output
			var sites []model.AllocSite
			for site := range sitesMap {
				sites = append(sites, site)
			}
			sort.Slice(sites, func(i, j int) bool {
				if sites[i].Position.Filename == sites[j].Position.Filename {
					return sites[i].Position.Line < sites[j].Position.Line
				}
				return sites[i].Position.Filename < sites[j].Position.Filename
			})

			for _, site := range sites {
				fmt.Printf("- %s %s\n", site.Position, site.Type)
			}
		}
		fmt.Println()
	}
}
