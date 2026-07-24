package matcher

import (
	"go/token"
	"testing"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestDetectDeadlocks(t *testing.T) {
	site := model.AllocSite{ID: 1, Type: "chan int", Position: token.Position{Line: 10}}
	varChan := model.ContextValue{Value: "var", Goroutine: "main"}
	writeOp := model.OpNode{OpType: model.OpWrite, Channel: varChan}
	readOp := model.OpNode{OpType: model.OpRead, Channel: varChan}
	closeOp := model.OpNode{OpType: model.OpClose, Channel: varChan}

	tests := []struct {
		name     string
		input    map[model.AllocSite]map[model.GoroutineID][]model.TraceNode
		expected []string
	}{
		{
			name: "Safe: 1 write, 1 read in different goroutines",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"main":   {writeOp},
					"worker": {readOp},
				},
			},
			expected: nil,
		},
		{
			name: "Rule 1: Single goroutine blocked (write only)",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"main": {writeOp},
				},
			},
			expected: []string{"Symbolic Execution"},
		},
		{
			name: "Rule 2: Count Mismatch (2 writes, 1 read)",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"main":   {writeOp, writeOp},
					"worker": {readOp},
				},
			},
			expected: []string{"Symbolic Execution"},
		},
		{
			name: "Rule 2: Order Mismatch (Both wait to write first)",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"G1": {writeOp, readOp},
					"G2": {writeOp, readOp},
				},
			},
			expected: []string{"Symbolic Execution"},
		},
		{
			name: "Rule 3: Forgotten Close (range without close)",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"main": {
						model.RangeNode{Channel: varChan},
					},
					"worker": {writeOp},
				},
			},
			expected: []string{"Symbolic Execution"},
		},
		{
			name: "Safe: Range with close",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"main": {
						model.RangeNode{Channel: varChan},
					},
					"worker": {writeOp, closeOp},
				},
			},
			expected: nil,
		},
		{
			name: "May-Deadlock: Loop exits early",
			input: map[model.AllocSite]map[model.GoroutineID][]model.TraceNode{
				site: {
					"main": {
						model.LoopNode{Bounds: "*", Body: []model.TraceNode{writeOp}},
					},
					"worker": {readOp},
				},
			},
			expected: []string{"Symbolic Execution"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reports := DetectDeadlocks(tc.input)
			if tc.expected == nil {
				assert.Empty(t, reports)
			} else {
				assert.Len(t, reports, len(tc.expected))
				for i, expStr := range tc.expected {
					assert.Contains(t, reports[i].Reason, expStr)
				}
			}
		})
	}
}
