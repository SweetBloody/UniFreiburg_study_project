package analysis

import (
	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/model"
)

// Solve runs the work-list algorithm to propagate channel flows
func Solve(state model.State, constraints []model.Constraint) {
	outEdges := make(map[model.ValueID][]model.ValueID)

	// Initialize worklist
	var worklist []model.ValueID
	inWorklist := make(map[model.ValueID]bool)

	for _, c := range constraints {
		outEdges[c.Source] = append(outEdges[c.Source], c.Target)

		// If the source already has some sets, put it in the worklist
		if len(state[c.Source]) > 0 && !inWorklist[c.Source] {
			worklist = append(worklist, c.Source)
			inWorklist[c.Source] = true
		}
	}

	// Work-list loop
	for len(worklist) > 0 {
		// Pop front
		source := worklist[0]
		worklist = worklist[1:]
		inWorklist[source] = false

		sourceSet := state[source]
		if len(sourceSet) == 0 {
			continue
		}

		// Propagate to all targets
		for _, target := range outEdges[source] {
			changed := false

			if state[target] == nil {
				state[target] = make(map[model.AllocSite]struct{})
			}
			targetSet := state[target]

			// Union operation
			for site := range sourceSet {
				if _, exists := targetSet[site]; !exists {
					targetSet[site] = struct{}{}
					changed = true
				}
			}

			// If changed, add target to worklist
			if changed && !inWorklist[target] {
				worklist = append(worklist, target)
				inWorklist[target] = true
			}
		}
	}
}
