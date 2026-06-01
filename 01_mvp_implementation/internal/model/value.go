package model

import (
	"fmt"

	"golang.org/x/tools/go/ssa"
)

// ValueID represents an SSA value, function parameter, or function return value
type ValueID string

// MakeValueID generates a reasonably unique string identifier for an ssa.Value
func MakeValueID(v ssa.Value) ValueID {
	if p := v.Parent(); p != nil {
		return ValueID(fmt.Sprintf("%s.%s", p.String(), v.Name()))
	}
	return ValueID(v.String())
}
