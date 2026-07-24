package model

// Constraint means the target state includes everything from the source state
type Constraint struct {
	Source ContextValue
	Target ContextValue
}

// State maps each contextual value to the set of possible channel allocation sites
type State map[ContextValue]map[AllocSite]struct{}

func NewState() State {
	return make(State)
}

type OpType string

const (
	OpRead  OpType = "READ"
	OpWrite OpType = "WRITE"
	OpClose OpType = "CLOSE"
)
