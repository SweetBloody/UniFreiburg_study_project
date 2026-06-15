package symbolic

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/SweetBloody/UniFreiburg_study_project/chanflow/03_symbolic_trace_implementation/internal/model"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type Builder struct {
	State   model.State
	visited map[string]bool
}

func NewBuilder(state model.State) *Builder {
	return &Builder{
		State:   state,
		visited: make(map[string]bool),
	}
}

func (b *Builder) Build(prog *ssa.Program) []model.TraceNode {
	cg := static.CallGraph(prog)

	var mainFunc *ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Name() == "main" && fn.Pkg != nil && fn.Pkg.Pkg.Name() == "main" {
			mainFunc = fn
			break
		}
	}

	if mainFunc == nil {
		return nil
	}

	mainNode := cg.Nodes[mainFunc]
	if mainNode == nil {
		return nil
	}

	return b.traverse(mainNode, model.GoroutineID("main"))
}

func (b *Builder) traverse(node *callgraph.Node, gID model.GoroutineID) []model.TraceNode {
	fn := node.Func
	if fn == nil || fn.Syntax() == nil {
		return nil
	}

	visitKey := fmt.Sprintf("%s@%s", fn.String(), gID)
	if b.visited[visitKey] {
		return []model.TraceNode{model.CallNode{FuncName: fn.Name()}}
	}
	b.visited[visitKey] = true

	rootNodes := b.traverseASTNode(fn.Syntax(), fn, node, gID)

	return rootNodes
}

func (b *Builder) traverseASTNode(n ast.Node, fn *ssa.Function, cgNode *callgraph.Node, gID model.GoroutineID) []model.TraceNode {
	if n == nil {
		return nil
	}
	var nodes []model.TraceNode

	ast.Inspect(n, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		switch v := node.(type) {
		case *ast.FuncLit:
			return false

		case *ast.ForStmt:
			body := b.traverseASTNode(v.Body, fn, cgNode, gID)
			bounds := "*"

			// Attempt to extract explicit integer bounds (e.g., i < 5)
			if v.Cond != nil {
				if binExpr, ok := v.Cond.(*ast.BinaryExpr); ok {
					if binExpr.Op == token.LSS || binExpr.Op == token.LEQ {
						if lit, ok := binExpr.Y.(*ast.BasicLit); ok && lit.Kind == token.INT {
							bounds = lit.Value
						}
					}
				}
			}

			if len(body) > 0 {
				nodes = append(nodes, model.LoopNode{Bounds: bounds, Body: body})
			}
			return false

		case *ast.RangeStmt:
			body := b.traverseASTNode(v.Body, fn, cgNode, gID)

			// A range over a channel implicitly reads from it on every iteration
			val, _ := fn.ValueForExpr(v.X)
			if val != nil {
				ctxVal := model.ContextValue{Value: model.MakeValueID(val), Goroutine: gID}
				if _, ok := b.State[ctxVal]; ok {
					body = append([]model.TraceNode{model.OpNode{OpType: model.OpRead, Channel: ctxVal}}, body...)
				}
			}

			if len(body) > 0 {
				nodes = append(nodes, model.LoopNode{Bounds: "*", Body: body})
			}
			return false

		case *ast.IfStmt:
			thenBody := b.traverseASTNode(v.Body, fn, cgNode, gID)
			var elseBody []model.TraceNode
			if v.Else != nil {
				elseBody = b.traverseASTNode(v.Else, fn, cgNode, gID)
			}
			if len(thenBody) > 0 || len(elseBody) > 0 {
				nodes = append(nodes, model.IfNode{Condition: "cond", Then: thenBody, Else: elseBody})
			}
			return false

		case *ast.SendStmt:
			// Use ValueForExpr to link AST to SSA
			val, _ := fn.ValueForExpr(v.Chan)
			if val != nil {
				ctxVal := model.ContextValue{Value: model.MakeValueID(val), Goroutine: gID}
				// Verify if it points to any channel
				if _, ok := b.State[ctxVal]; ok {
					nodes = append(nodes, model.OpNode{OpType: model.OpWrite, Channel: ctxVal})
				}
			}

		case *ast.UnaryExpr:
			// Reading
			if v.Op == token.ARROW {
				val, _ := fn.ValueForExpr(v.X)
				if val != nil {
					ctxVal := model.ContextValue{Value: model.MakeValueID(val), Goroutine: gID}
					if _, ok := b.State[ctxVal]; ok {
						nodes = append(nodes, model.OpNode{OpType: model.OpRead, Channel: ctxVal})
					}
				}
			}

		case *ast.GoStmt:
			// A spawned goroutine
			pos := fn.Prog.Fset.Position(v.Pos())
			matched := false
			for _, edge := range cgNode.Out {
				if edge.Site.Pos() == v.Pos() {
					matched = true
					if _, isGo := edge.Site.(*ssa.Go); isGo {
						nextGID := model.GoroutineID(fmt.Sprintf("%s:%d", pos.Filename, pos.Line))
						childRoot := b.traverse(edge.Callee, nextGID)
						nodes = append(nodes, childRoot...)
					}
					break
				}
			}
			if !matched {
				for _, edge := range cgNode.Out {
					edgePos := fn.Prog.Fset.Position(edge.Site.Pos())
					if edgePos.Line == pos.Line {
						if _, isGo := edge.Site.(*ssa.Go); isGo {
							nextGID := model.GoroutineID(fmt.Sprintf("%s:%d", pos.Filename, pos.Line))
							childRoot := b.traverse(edge.Callee, nextGID)
							nodes = append(nodes, childRoot...)
						}
						break
					}
				}
			}
			return false

		case *ast.CallExpr:
			// close() call
			if ident, ok := v.Fun.(*ast.Ident); ok && ident.Name == "close" && len(v.Args) > 0 {
				val, _ := fn.ValueForExpr(v.Args[0])
				if val != nil {
					ctxVal := model.ContextValue{Value: model.MakeValueID(val), Goroutine: gID}
					if _, ok := b.State[ctxVal]; ok {
						nodes = append(nodes, model.OpNode{OpType: model.OpClose, Channel: ctxVal})
					}
				}
				return false
			}

			// function call
			matched := false
			for _, edge := range cgNode.Out {
				if edge.Site.Pos() == v.Pos() {
					matched = true
					if _, isGo := edge.Site.(*ssa.Go); !isGo {
						// Inline the function call trace!
						childTrace := b.traverse(edge.Callee, gID)
						nodes = append(nodes, childTrace...)
					}
					break
				}
			}
			if !matched {
				// If exact Pos() fails
				pos := fn.Prog.Fset.Position(v.Pos())
				for _, edge := range cgNode.Out {
					edgePos := fn.Prog.Fset.Position(edge.Site.Pos())
					if edgePos.Line == pos.Line {
						if _, isGo := edge.Site.(*ssa.Go); !isGo {
							childTrace := b.traverse(edge.Callee, gID)
							nodes = append(nodes, childTrace...)
						}
						break
					}
				}
			}
			return false
		}

		return true
	})

	return nodes
}

// ProjectAll generates projected traces for all discovered channels
func (b *Builder) ProjectAll(nodes []model.TraceNode) map[model.AllocSite][]model.TraceNode {
	// Extract all unique channels
	allocSites := make(map[model.AllocSite]struct{})
	for _, sites := range b.State {
		for site := range sites {
			allocSites[site] = struct{}{}
		}
	}

	// Project the trace for each channel
	result := make(map[model.AllocSite][]model.TraceNode)
	for site := range allocSites {
		projected := b.projectTrace(nodes, site)
		result[site] = projected
	}

	return result
}

// projectTrace filters a unified trace to only include operations on the given AllocSite,
// pruning any loops or branches that become empty.
func (b *Builder) projectTrace(nodes []model.TraceNode, targetAlloc model.AllocSite) []model.TraceNode {
	var result []model.TraceNode

	for _, n := range nodes {
		switch v := n.(type) {
		case model.OpNode:
			// Check if this operation's channel points to targetAlloc
			if sites, ok := b.State[v.Channel]; ok {
				if _, matches := sites[targetAlloc]; matches {
					result = append(result, v)
				}
			}

		case model.LoopNode:
			body := b.projectTrace(v.Body, targetAlloc)
			if len(body) > 0 {
				result = append(result, model.LoopNode{Bounds: v.Bounds, Body: body})
			}

		case model.IfNode:
			thenBody := b.projectTrace(v.Then, targetAlloc)
			var elseBody []model.TraceNode
			if v.Else != nil {
				elseBody = b.projectTrace(v.Else, targetAlloc)
			}
			if len(thenBody) > 0 || len(elseBody) > 0 {
				result = append(result, model.IfNode{Condition: v.Condition, Then: thenBody, Else: elseBody})
			}

		case model.CallNode:
			// Recursion func call
			result = append(result, v)
		}
	}

	return result
}
