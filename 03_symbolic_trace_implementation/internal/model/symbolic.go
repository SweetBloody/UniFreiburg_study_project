package model

import (
	"fmt"
	"strings"
)

// TraceNode represents an element in the symbolic execution trace
type TraceNode interface {
	String() string
}

// OpNode represents a single channel operation (! for Write, ? for Read, X for Close)
type OpNode struct {
	OpType  OpType
	Channel ContextValue
}

func (n OpNode) String() string {
	switch n.OpType {
	case OpWrite:
		return "!"
	case OpRead:
		return "?"
	case OpClose:
		return "X"
	}
	return "UNKNOWN"
}

// LoopNode represents a loop in the code containing a sequence of operations
type LoopNode struct {
	Bounds string // "*" for undefined, "n" for value
	Body   []TraceNode
}

func (n LoopNode) String() string {
	var bodyStrs []string
	for _, b := range n.Body {
		bodyStrs = append(bodyStrs, b.String())
	}
	return fmt.Sprintf("loop(%s, [%s])", n.Bounds, strings.Join(bodyStrs, ", "))
}

// IfNode represents a conditional branch
type IfNode struct {
	Condition string
	Then      []TraceNode
	Else      []TraceNode
}

func (n IfNode) String() string {
	var thenStrs []string
	for _, b := range n.Then {
		thenStrs = append(thenStrs, b.String())
	}

	var elseStrs []string
	for _, b := range n.Else {
		elseStrs = append(elseStrs, b.String())
	}

	thenStr := "[" + strings.Join(thenStrs, ", ") + "]"
	elseStr := "[" + strings.Join(elseStrs, ", ") + "]"

	return fmt.Sprintf("if(%s, %s, %s)", n.Condition, thenStr, elseStr)
}

// CallNode represents a function call (in case of recursion)
type CallNode struct {
	FuncName string
}

func (n CallNode) String() string {
	return fmt.Sprintf("call(%s)", n.FuncName)
}
