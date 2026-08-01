package matcher

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/model"
)

// MachineState represents the state of a single channel's projected traces
type MachineState struct {
	Goroutines map[model.GoroutineID][]model.TraceNode
	Closed     bool
}

// cloneState creates a deep copy of the MachineState
func cloneState(s MachineState) MachineState {
	clone := MachineState{
		Goroutines: make(map[model.GoroutineID][]model.TraceNode),
		Closed:     s.Closed,
	}
	for gID, trace := range s.Goroutines {
		if len(trace) > 0 {
			// shallow copy of the slice (pointers to TraceNodes remain the same)
			tCopy := make([]model.TraceNode, len(trace))
			copy(tCopy, trace)
			clone.Goroutines[gID] = tCopy
		}
	}
	return clone
}

// HashState returns a deterministic string representation for visited set
func HashState(s MachineState) string {
	var gIDs []string
	for k, trace := range s.Goroutines {
		if len(trace) > 0 {
			gIDs = append(gIDs, string(k))
		}
	}
	sort.Strings(gIDs)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("C:%v;", s.Closed))
	for _, gID := range gIDs {
		b.WriteString(gID)
		b.WriteString(":[")
		for _, n := range s.Goroutines[model.GoroutineID(gID)] {
			b.WriteString(n.String())
			b.WriteString(",")
		}
		b.WriteString("];")
	}
	return b.String()
}

// NextStates computes all valid subsequent states from the current state
func NextStates(s MachineState) []MachineState {
	var nextStates []MachineState

	// 1. Normalization (Local non-blocking transitions)
	// If any goroutine has a control node at head, expand it and return (one step at a time)
	for gID, trace := range s.Goroutines {
		if len(trace) == 0 {
			continue
		}
		head := trace[0]
		tail := trace[1:]

		switch v := head.(type) {
		case model.IfNode:
			s1 := cloneState(s)
			s1.Goroutines[gID] = append(v.Then, tail...)
			nextStates = append(nextStates, s1)

			s2 := cloneState(s)
			s2.Goroutines[gID] = append(v.Else, tail...)
			nextStates = append(nextStates, s2)
			return nextStates

		case model.LoopNode:
			// If Bounds is a concrete integer N >= 0, unroll deterministically
			if n, err := strconv.Atoi(v.Bounds); err == nil && n >= 0 && n <= 100 {
				if n == 0 {
					s1 := cloneState(s)
					s1.Goroutines[gID] = tail
					nextStates = append(nextStates, s1)
					return nextStates
				}
				s1 := cloneState(s)
				newTrace := make([]model.TraceNode, 0, len(v.Body)+1+len(tail))
				newTrace = append(newTrace, v.Body...)
				if n > 1 {
					newTrace = append(newTrace, model.LoopNode{
						Bounds: strconv.Itoa(n - 1),
						Body:   v.Body,
					})
				}
				newTrace = append(newTrace, tail...)
				s1.Goroutines[gID] = newTrace
				nextStates = append(nextStates, s1)
				return nextStates
			}

			// Branch 1: Exit loop
			s1 := cloneState(s)
			s1.Goroutines[gID] = tail
			nextStates = append(nextStates, s1)

			// Branch 2: Do one iteration and repeat
			s2 := cloneState(s)
			newTrace := make([]model.TraceNode, 0, len(v.Body)+1+len(tail))
			newTrace = append(newTrace, v.Body...)
			newTrace = append(newTrace, head) // put the loop back
			newTrace = append(newTrace, tail...)
			s2.Goroutines[gID] = newTrace
			nextStates = append(nextStates, s2)

			return nextStates

		case model.CallNode:
			// Local non-blocking transition, just ignore the call node
			s1 := cloneState(s)
			s1.Goroutines[gID] = tail
			nextStates = append(nextStates, s1)
			return nextStates
		}
	}

	// 2. Communication transitions (All heads are now OpNode or RangeNode)
	heads := make(map[model.GoroutineID]model.TraceNode)
	for gID, trace := range s.Goroutines {
		if len(trace) > 0 {
			heads[gID] = trace[0]
		}
	}

	// Single-goroutine transitions (Close, ClosedReceive, ClosedRange, PanicSend)
	for gID, head := range heads {
		if op, ok := head.(model.OpNode); ok {
			if op.OpType == model.OpClose {
				if !s.Closed {
					ns := cloneState(s)
					ns.Closed = true
					ns.Goroutines[gID] = ns.Goroutines[gID][1:]
					nextStates = append(nextStates, ns)
				} else {
					// Double close panic - terminates this goroutine
					ns := cloneState(s)
					ns.Goroutines[gID] = nil
					nextStates = append(nextStates, ns)
				}
			} else if op.OpType == model.OpRead && s.Closed {
				// Receive from closed channel
				ns := cloneState(s)
				ns.Goroutines[gID] = ns.Goroutines[gID][1:]
				nextStates = append(nextStates, ns)
			} else if op.OpType == model.OpWrite && s.Closed {
				// Send to closed channel -> PANIC
				ns := cloneState(s)
				ns.Goroutines[gID] = nil
				nextStates = append(nextStates, ns)
			}
		} else if _, ok := head.(model.RangeNode); ok {
			if s.Closed {
				// Range over closed channel finishes
				ns := cloneState(s)
				ns.Goroutines[gID] = ns.Goroutines[gID][1:]
				nextStates = append(nextStates, ns)
			}
		}
	}

	// Pairwise communication transitions
	if !s.Closed {
		// Convert map to slice for predictable iteration
		var gIDs []model.GoroutineID
		for gID := range heads {
			gIDs = append(gIDs, gID)
		}

		for i := 0; i < len(gIDs); i++ {
			for j := i + 1; j < len(gIDs); j++ {
				g1 := gIDs[i]
				g2 := gIDs[j]
				head1 := heads[g1]
				head2 := heads[g2]

				op1, isOp1 := head1.(model.OpNode)
				rng1, isRng1 := head1.(model.RangeNode)
				op2, isOp2 := head2.(model.OpNode)
				rng2, isRng2 := head2.(model.RangeNode)

				if isOp1 && isOp2 {
					if (op1.OpType == model.OpWrite && op2.OpType == model.OpRead) ||
						(op1.OpType == model.OpRead && op2.OpType == model.OpWrite) {
						ns := cloneState(s)
						ns.Goroutines[g1] = ns.Goroutines[g1][1:]
						ns.Goroutines[g2] = ns.Goroutines[g2][1:]
						nextStates = append(nextStates, ns)
					}
				} else if isOp1 && isRng2 && op1.OpType == model.OpWrite {
					ns := cloneState(s)
					ns.Goroutines[g1] = ns.Goroutines[g1][1:]
					newG2 := make([]model.TraceNode, 0, len(rng2.Body)+1+len(ns.Goroutines[g2][1:]))
					newG2 = append(newG2, rng2.Body...)
					newG2 = append(newG2, head2)
					newG2 = append(newG2, ns.Goroutines[g2][1:]...)
					ns.Goroutines[g2] = newG2
					nextStates = append(nextStates, ns)
				} else if isRng1 && isOp2 && op2.OpType == model.OpWrite {
					ns := cloneState(s)
					ns.Goroutines[g2] = ns.Goroutines[g2][1:]
					newG1 := make([]model.TraceNode, 0, len(rng1.Body)+1+len(ns.Goroutines[g1][1:]))
					newG1 = append(newG1, rng1.Body...)
					newG1 = append(newG1, head1)
					newG1 = append(newG1, ns.Goroutines[g1][1:]...)
					ns.Goroutines[g1] = newG1
					nextStates = append(nextStates, ns)
				}
			}
		}
	}

	return nextStates
}
