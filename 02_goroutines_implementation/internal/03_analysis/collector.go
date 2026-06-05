package analysis

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/02_goroutines_implementation/internal/model"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Collector holds the analysis state (values -> allocations), constraints, and collected operations
type Collector struct {
	State       model.State
	Constraints []model.Constraint
	Operations  []model.ChanOp
	allocID     int
	visited     map[string]bool
}

func NewCollector() *Collector {
	return &Collector{
		State:       model.NewState(),
		Constraints: make([]model.Constraint, 0),
		Operations:  make([]model.ChanOp, 0),
		allocID:     1,
		visited:     make(map[string]bool),
	}
}

// Collect traverses the Call Graph starting from "main"
func (c *Collector) Collect(prog *ssa.Program) {
	cg := static.CallGraph(prog)

	var mainFunc *ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Name() == "main" && fn.Pkg != nil && fn.Pkg.Pkg.Name() == "main" {
			mainFunc = fn
			break
		}
	}

	if mainFunc == nil {
		return
	}

	mainNode := cg.Nodes[mainFunc]
	if mainNode == nil {
		return
	}

	c.traverse(mainNode, model.GoroutineID("main"))
}

func (c *Collector) traverse(node *callgraph.Node, gID model.GoroutineID) {
	fn := node.Func
	if fn == nil {
		return
	}

	// Create a unique visit key based on function and goroutine to prevent infinite loops
	visitKey := fmt.Sprintf("%s@%s", fn.String(), gID)
	if c.visited[visitKey] {
		return
	}
	c.visited[visitKey] = true

	// Scan instructions
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			c.processInstruction(instr, gID, fn.Prog.Fset)
		}
	}

	// Traverse outgoing call graph edges
	for _, edge := range node.Out {
		calleeNode := edge.Callee
		if calleeNode == nil || calleeNode.Func == nil || edge.Site == nil {
			continue
		}

		nextGID := gID
		// If it's a 'go' statement, create a new GoroutineID
		if _, isGo := edge.Site.(*ssa.Go); isGo {
			pos := fn.Prog.Fset.Position(edge.Site.Pos())
			nextGID = model.GoroutineID(fmt.Sprintf("%s:%d", pos.Filename, pos.Line))
		}

		// Match arguments to parameters to generate constraints (source var -> target var)
		c.matchArgumentConstraints(edge, calleeNode.Func, gID, nextGID)

		// return value - var
		if call, ok := edge.Site.(*ssa.Call); ok {
			c.matchReturnConstraints(call, calleeNode.Func, gID, nextGID)
		}

		c.traverse(calleeNode, nextGID)
	}
}

func (c *Collector) processInstruction(instr ssa.Instruction, gID model.GoroutineID, fset *token.FileSet) {
	switch instr := instr.(type) {
	case *ssa.MakeChan:
		site := model.AllocSite{
			ID:       c.allocID,
			Position: fset.Position(instr.Pos()),
			Type:     instr.Type().String(),
		}
		c.allocID++

		val := model.ContextValue{Value: model.MakeValueID(instr), Goroutine: gID}
		if c.State[val] == nil {
			c.State[val] = make(map[model.AllocSite]struct{})
		}
		c.State[val][site] = struct{}{}

	case *ssa.Send:
		val := model.ContextValue{Value: model.MakeValueID(instr.Chan), Goroutine: gID}
		c.Operations = append(c.Operations, model.ChanOp{
			Type:       model.OpWrite,
			ChannelVar: val,
			Position:   fset.Position(instr.Pos()).String(),
		})

	case *ssa.UnOp:
		if instr.Op == token.ARROW {
			val := model.ContextValue{Value: model.MakeValueID(instr.X), Goroutine: gID}
			c.Operations = append(c.Operations, model.ChanOp{
				Type:       model.OpRead,
				ChannelVar: val,
				Position:   fset.Position(instr.Pos()).String(),
			})
		}

	case *ssa.Call:
		if builtin, ok := instr.Call.Value.(*ssa.Builtin); ok && builtin.Name() == "close" {
			if len(instr.Call.Args) > 0 {
				val := model.ContextValue{Value: model.MakeValueID(instr.Call.Args[0]), Goroutine: gID}
				c.Operations = append(c.Operations, model.ChanOp{
					Type:       model.OpClose,
					ChannelVar: val,
					Position:   fset.Position(instr.Pos()).String(),
				})
			}
		}

	case *ssa.Phi:
		if isChanType(instr.Type()) {
			targetVal := model.ContextValue{Value: model.MakeValueID(instr), Goroutine: gID}
			for _, edgeVal := range instr.Edges {
				sourceVal := model.ContextValue{Value: model.MakeValueID(edgeVal), Goroutine: gID}
				c.Constraints = append(c.Constraints, model.Constraint{
					Source: sourceVal,
					Target: targetVal,
				})
			}
		}
	}
}

// matchArgumentConstraints links arguments passed in a function call to the function's formal parameters
func (c *Collector) matchArgumentConstraints(edge *callgraph.Edge, callee *ssa.Function, gID, nextGID model.GoroutineID) {
	args := edge.Site.Common().Args
	params := callee.Params

	for i, arg := range args {
		if i >= len(params) {
			continue
		}

		param := params[i]
		if !isChanType(param.Type()) {
			continue
		}

		sourceVal := model.ContextValue{Value: model.MakeValueID(arg), Goroutine: gID}
		targetVal := model.ContextValue{Value: model.MakeValueID(param), Goroutine: nextGID}

		c.Constraints = append(c.Constraints, model.Constraint{
			Source: sourceVal,
			Target: targetVal,
		})
	}
}

// matchReturnConstraints scans the callee function for Return instructions and links them to the Caller's variable
func (c *Collector) matchReturnConstraints(callVal ssa.Value, callee *ssa.Function, gID, nextGID model.GoroutineID) {
	if callVal == nil || callee == nil || callee.Blocks == nil {
		return
	}

	for _, block := range callee.Blocks {
		for _, instr := range block.Instrs {
			ret, isRet := instr.(*ssa.Return)
			if !isRet {
				continue
			}

			if _, isTuple := callVal.Type().(*types.Tuple); isTuple {
				c.handleTupleReturn(callVal, ret, gID, nextGID)
			} else if len(ret.Results) == 1 {
				c.handleSingleReturn(callVal, ret, gID, nextGID)
			}
		}
	}
}

// handleTupleReturn processes multiple returns by linking extracted elements to the corresponding return results
func (c *Collector) handleTupleReturn(callVal ssa.Value, ret *ssa.Return, gID, nextGID model.GoroutineID) {
	referrers := callVal.Referrers()
	if referrers == nil {
		return
	}

	for _, ref := range *referrers {
		extract, isExtract := ref.(*ssa.Extract)
		if !isExtract {
			continue
		}

		res := ret.Results[extract.Index]
		if !isChanType(res.Type()) {
			continue
		}

		sourceVal := model.ContextValue{Value: model.MakeValueID(res), Goroutine: nextGID}
		targetVal := model.ContextValue{Value: model.MakeValueID(extract), Goroutine: gID}
		c.Constraints = append(c.Constraints, model.Constraint{
			Source: sourceVal, Target: targetVal,
		})
	}
}

// handleSingleReturn processes a single return by linking the return result directly to the call value
func (c *Collector) handleSingleReturn(callVal ssa.Value, ret *ssa.Return, gID, nextGID model.GoroutineID) {
	res := ret.Results[0]
	if !isChanType(res.Type()) {
		return
	}

	sourceVal := model.ContextValue{Value: model.MakeValueID(res), Goroutine: nextGID}
	targetVal := model.ContextValue{Value: model.MakeValueID(callVal), Goroutine: gID}
	c.Constraints = append(c.Constraints, model.Constraint{
		Source: sourceVal, Target: targetVal,
	})
}

// isChanType checks if the given type is a channel or a directional channel
func isChanType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Chan)
	return ok
}
