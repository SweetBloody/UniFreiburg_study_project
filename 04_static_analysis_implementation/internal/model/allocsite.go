package model

import "go/token"

// AllocSite represents one make(chan T) instruction
type AllocSite struct {
	ID       int
	Position token.Position
	Type     string
}
