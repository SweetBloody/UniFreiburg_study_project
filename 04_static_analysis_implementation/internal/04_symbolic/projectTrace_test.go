package symbolic

import (
	"go/token"
	"testing"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/04_static_analysis_implementation/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestProjectTrace(t *testing.T) {
	varA := model.ContextValue{Value: "varA", Goroutine: "main"}
	varB := model.ContextValue{Value: "varB", Goroutine: "main"}
	//varC := model.ContextValue{Value: "varC", Goroutine: "main"}
	//varD := model.ContextValue{Value: "varD", Goroutine: "main"}

	site1 := model.AllocSite{ID: 1, Type: "chan int", Position: token.Position{Line: 7}}
	site2 := model.AllocSite{ID: 2, Type: "chan int", Position: token.Position{Line: 12}}
	//site3 := model.AllocSite{ID: 3, Type: "chan int", Position: token.Position{Line: 15}}

	fakeState := model.State{
		varA: {site1: struct{}{}},
		varB: {site2: struct{}{}},
	}

	tests := []struct {
		name     string
		input    []model.TraceNode
		expected []model.TraceNode
	}{
		{
			name: "Remove irrelevant operations",
			input: []model.TraceNode{
				model.OpNode{OpType: model.OpWrite, Channel: varA},
				model.OpNode{OpType: model.OpRead, Channel: varB},
			},
			expected: []model.TraceNode{
				model.OpNode{OpType: model.OpWrite, Channel: varA},
			},
		},
		{
			name: "Remove empty loop",
			input: []model.TraceNode{
				model.LoopNode{
					Bounds: "*",
					Body: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varB},
						model.OpNode{OpType: model.OpRead, Channel: varB},
					},
				},
				model.OpNode{OpType: model.OpWrite, Channel: varA},
				model.OpNode{OpType: model.OpRead, Channel: varB},
			},
			expected: []model.TraceNode{
				model.OpNode{OpType: model.OpWrite, Channel: varA},
			},
		},
		{
			name: "RangeNode(varB) -> Loop for varA",
			input: []model.TraceNode{
				model.RangeNode{
					Channel: varB,
					Body: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varA},
						model.OpNode{OpType: model.OpRead, Channel: varB},
					},
				},
				model.OpNode{OpType: model.OpWrite, Channel: varA},
				model.OpNode{OpType: model.OpRead, Channel: varB},
			},
			expected: []model.TraceNode{
				model.LoopNode{
					Bounds: "*",
					Body: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varA},
					},
				},
				model.OpNode{OpType: model.OpWrite, Channel: varA},
			},
		},
		{
			name: "RangeNode(varA) -> RangeNode for varA",
			input: []model.TraceNode{
				model.RangeNode{
					Channel: varA,
					Body: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varA},
						model.OpNode{OpType: model.OpRead, Channel: varB},
					},
				},
				model.OpNode{OpType: model.OpWrite, Channel: varA},
				model.OpNode{OpType: model.OpRead, Channel: varB},
			},
			expected: []model.TraceNode{
				model.RangeNode{
					Channel: varA,
					Body: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varA},
					},
				},
				model.OpNode{OpType: model.OpWrite, Channel: varA},
			},
		},
		{
			name: "Remove IfNode if empty",
			input: []model.TraceNode{
				model.IfNode{
					Condition: "true",
					Then: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varB},
						model.OpNode{OpType: model.OpRead, Channel: varB},
					},
					Else: []model.TraceNode{
						model.OpNode{OpType: model.OpWrite, Channel: varB},
						model.OpNode{OpType: model.OpRead, Channel: varB},
					},
				},
				model.OpNode{OpType: model.OpWrite, Channel: varA},
				model.OpNode{OpType: model.OpRead, Channel: varB},
			},
			expected: []model.TraceNode{
				model.OpNode{OpType: model.OpWrite, Channel: varA},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := NewBuilder(fakeState)
			actual := b.projectTrace(test.input, site1)

			if test.expected == nil {
				assert.Empty(t, actual)
			} else {
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}
