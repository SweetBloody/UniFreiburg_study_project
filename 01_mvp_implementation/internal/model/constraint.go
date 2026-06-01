package model

// Constraint means State[Target] ⊇ State[Source]
type Constraint struct {
	Source ValueID
	Target ValueID
}

// State maps each analysis value to the set of possible channel allocation sites
type State map[ValueID]map[AllocSite]struct{}

// NewState creates and returns a new State
func NewState() State {
	return make(State)
}
