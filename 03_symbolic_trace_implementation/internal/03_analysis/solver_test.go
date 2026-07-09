package analysis_test

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	analysis "github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/03_analysis"
	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/model"
)

func TestSolver(t *testing.T) {
	varA := model.ContextValue{Value: "varA", Goroutine: "main"}
	varB := model.ContextValue{Value: "varB", Goroutine: "main"}
	varC := model.ContextValue{Value: "varC", Goroutine: "main"}
	varD := model.ContextValue{Value: "varD", Goroutine: "main"}

	site1 := model.AllocSite{ID: 1, Type: "chan int", Position: token.Position{Line: 7}}
	site2 := model.AllocSite{ID: 2, Type: "chan int", Position: token.Position{Line: 12}}
	//site3 := model.AllocSite{ID: 3, Type: "chan int", Position: token.Position{Line: 15}}

	tests := []struct {
		name          string
		state         model.State
		constraints   []model.Constraint
		expectedState model.State
	}{
		{
			name: "Simple varA -> varB",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varB},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varB: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
		},
		{
			name: "Transfer varA -> varB, varB -> varC",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varB},
				{Source: varB, Target: varC},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varB: map[model.AllocSite]struct{}{
					site1: {},
				},
				varC: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
		},
		{
			name: "If SMTH varC = varA, else varC = varB",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varB: map[model.AllocSite]struct{}{
					site2: {},
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varC},
				{Source: varB, Target: varC},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varB: map[model.AllocSite]struct{}{
					site2: {},
				},
				varC: map[model.AllocSite]struct{}{
					site1: {},
					site2: {},
				},
			},
		},
		{
			name: "varA -> varB -> varC -> varA",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varB},
				{Source: varB, Target: varC},
				{Source: varC, Target: varA},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varB: map[model.AllocSite]struct{}{
					site1: {},
				},
				varC: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
		},
		{
			name: "Cascade",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varC: map[model.AllocSite]struct{}{
					site2: {},
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varB},
				{Source: varB, Target: varC},
				{Source: varC, Target: varD},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
				varB: map[model.AllocSite]struct{}{
					site1: {},
				},
				varC: map[model.AllocSite]struct{}{
					site1: {},
				    site2: {},
				},
				varD: map[model.AllocSite]struct{}{
					site1: {},
				    site2: {},
				},
			},
		},
		{
			name: "Empty site",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
				},
				varB: map[model.AllocSite]struct{}{
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varB},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
				},
				varB: map[model.AllocSite]struct{}{
				},
			},
		},
		{
			name: "Self loop",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
			constraints: []model.Constraint{
				{Source: varA, Target: varA},
			},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
		},
		{
			name: "No constraints",
			state: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
			constraints: []model.Constraint{},
			expectedState: model.State{
				varA: map[model.AllocSite]struct{}{
					site1: {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis.Solve(test.state, test.constraints)
			assert.Equal(t, test.expectedState, test.state)
		})
	}
}
