package model

import (
	"fmt"

	"golang.org/x/tools/go/ssa"
)

// ValueID represents an SSA value, function parameter, or function return value
type ValueID string

// GoroutineID represents the identity of a goroutine
type GoroutineID string

// ContextValue pairs a ValueID with its executing GoroutineID
type ContextValue struct {
	Value     ValueID
	Goroutine GoroutineID
}

// MakeValueID generates a reasonably unique string identifier for an ssa.Value
func MakeValueID(v ssa.Value) ValueID {
	if p := v.Parent(); p != nil {
		return ValueID(fmt.Sprintf("%s.%s", p.String(), v.Name()))
	}
	return ValueID(v.String())
}

// String returns a readable representation of the contextual value
func (cv ContextValue) String() string {
	return fmt.Sprintf("%s@%s", cv.Value, cv.Goroutine)
}
