package matcher

import (
	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/model"
)

// Verdict represents the outcome of the BFS traversal for a channel
type Verdict int

const (
	VerdictSafe Verdict = iota
	VerdictDeadlock
	VerdictPanic
)

// DeadlockReport holds information about a detected deadlock or panic
type DeadlockReport struct {
	Channel model.AllocSite
	Reason  string
}

// DetectDeadlocks analyzes projected traces and finds deadlocks or panics using State Space Exploration
func DetectDeadlocks(projected map[model.AllocSite]map[model.GoroutineID][]model.TraceNode) []DeadlockReport {
	var reports []DeadlockReport

	for site, gMap := range projected {
		if len(gMap) == 0 {
			continue
		}

		initialState := MachineState{
			Goroutines: gMap,
			Closed:     false,
			Panicked:   false,
		}

		verdict := analyzeBFS(initialState)
		if verdict == VerdictDeadlock {
			reports = append(reports, DeadlockReport{
				Channel: site,
				Reason:  "Symbolic Execution (BFS): Found a terminal state where live goroutines are blocked waiting for communication (Deadlock or May-Deadlock).",
			})
		} else if verdict == VerdictPanic {
			reports = append(reports, DeadlockReport{
				Channel: site,
				Reason:  "Panic: Detected an attempt to send to or close an already closed channel.",
			})
		}
	}

	return reports
}

// analyzeBFS explores the state graph. Returns the Verdict for the channel
func analyzeBFS(initialState MachineState) Verdict {
	worklist := []MachineState{cloneState(initialState)}
	visited := make(map[string]bool)

	// To prevent infinite exploration of Loop(*), limit max states explored
	const maxStates = 1000
	statesExplored := 0

	for len(worklist) > 0 {
		state := worklist[0]
		worklist = worklist[1:]

		// If a state has already paniced, we immediately halt and report panic
		if state.Panicked {
			return VerdictPanic
		}

		h := HashState(state)
		if visited[h] {
			continue
		}
		visited[h] = true
		statesExplored++
		if statesExplored > maxStates {
			// Heuristic: if state space is huge and we haven't resolved, it's likely a non-terminating loop or may-deadlock
			return VerdictDeadlock
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
				return VerdictDeadlock
			}
		} else {
			worklist = append(worklist, nexts...)
		}
	}

	return VerdictSafe
}
