package analysis

import (
	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/internal/model"

	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/ssa"
)

// GenerateCallConstraints creates constraints for direct function calls
func GenerateCallConstraints(prog *ssa.Program) []model.Constraint {
	var constraints []model.Constraint

	cg := static.CallGraph(prog)

	// Traverse all nodes in the call graph
	for _, node := range cg.Nodes {
		for _, edge := range node.Out {
			caller := edge.Caller
			callee := edge.Callee

			if caller == nil || callee == nil || edge.Site == nil {
				continue
			}

			calleeFunc := callee.Func
			if calleeFunc == nil {
				continue
			}

			args := edge.Site.Common().Args
			params := calleeFunc.Params

			// Match actual arguments to formal parameters
			for i, arg := range args {
				if i < len(params) {
					param := params[i]
					if isChanType(param.Type()) {
						sourceID := model.MakeValueID(arg)
						targetID := model.MakeValueID(param)
						constraints = append(constraints, model.Constraint{
							Source: sourceID,
							Target: targetID,
						})
					}
				}
			}
		}
	}

	return constraints
}
