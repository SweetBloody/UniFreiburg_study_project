package analysis

import (
	"go/types"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/01_mvp_implementation/internal/model"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Collector holds the state and target parameters
type Collector struct {
	State            model.State
	TargetParameters []TargetParameter
}

type TargetParameter struct {
	Param *ssa.Parameter
	ID    model.ValueID
}

func NewCollector() *Collector {
	return &Collector{
		State:            model.NewState(),
		TargetParameters: make([]TargetParameter, 0),
	}
}

// Collect traverses the SSA program to initialize allocation sites and find target parameters
func (c *Collector) Collect(prog *ssa.Program) {
	allocID := 1
	for fn := range ssautil.AllFunctions(prog) {
		if fn == nil {
			continue
		}

		// Collect target parameters
		for _, param := range fn.Params {
			if isChanType(param.Type()) {
				id := model.MakeValueID(param)
				c.TargetParameters = append(c.TargetParameters, TargetParameter{
					Param: param,
					ID:    id,
				})
			}
		}

		// Collect allocation sites
		if fn.Blocks == nil {
			continue
		}

		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if makeChan, ok := instr.(*ssa.MakeChan); ok {
					site := model.AllocSite{
						ID:       allocID,
						Position: prog.Fset.Position(makeChan.Pos()),
						Type:     makeChan.Type().String(),
					}
					allocID++

					id := model.MakeValueID(makeChan)
					if c.State[id] == nil {
						c.State[id] = make(map[model.AllocSite]struct{})
					}
					c.State[id][site] = struct{}{}
				}
			}
		}
	}
}

// isChanType checks if the given type is a channel or a directional channel
func isChanType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Chan)
	return ok
}
