package matcher

import (
	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/model"
)

// DeadlockReport holds information about a detected deadlock
type DeadlockReport struct {
	Channel model.AllocSite
	Reason  string
}

// DetectDeadlocks analyzes projected traces and finds deadlocks using State Space Exploration
func DetectDeadlocks(projected map[model.AllocSite]map[model.GoroutineID][]model.TraceNode) []DeadlockReport {
	var reports []DeadlockReport

	for site, gMap := range projected {
		if len(gMap) == 0 {
			continue
		}

		initialState := MachineState{
			Goroutines: gMap,
			Closed:     false,
		}

		if isDeadlockedBFS(initialState) {
			reports = append(reports, DeadlockReport{
				Channel: site,
				Reason:  "Symbolic Execution (BFS): Found a terminal state where live goroutines are blocked waiting for communication (Deadlock or May-Deadlock).",
			})
		}
	}

	return reports
}

// isDeadlockedBFS explores the state graph. Returns true if a deadlock state is reachable.
func isDeadlockedBFS(initialState MachineState) bool {
	worklist := []MachineState{cloneState(initialState)}
	visited := make(map[string]bool)

	// To prevent infinite exploration of Loop(*), limit max states explored
	const maxStates = 1000
	statesExplored := 0

	for len(worklist) > 0 {
		state := worklist[0]
		worklist = worklist[1:]

		h := HashState(state)
		if visited[h] {
			continue
		}
		visited[h] = true
		statesExplored++
		if statesExplored > maxStates {
			// Heuristic: if state space is huge and we haven't resolved, it's likely a non-terminating loop or may-deadlock
			return true
		}

		nexts := NextStates(state)

		if len(nexts) == 0 {
			// Terminal state. Check if any goroutines are still blocked
			deadlocked := false
			for _, trace := range state.Goroutines {
				if len(trace) > 0 {
					deadlocked = true
					break
				}
			}
			if deadlocked {
				return true
			}
		} else {
			worklist = append(worklist, nexts...)
		}
	}

	return false
}
