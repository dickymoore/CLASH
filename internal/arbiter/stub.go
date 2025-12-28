package arbiter

import (
	"clash/internal/classifier"
)

// Input captures minimal data for arbiter triage.
type Input struct {
	Command string
	Signals []string
	Reasons []string
}

// Decision mirrors the ladder but can only tighten.
type Decision struct {
	Decision classifier.DecisionType
	Reason   string
}

// Decide returns a conservative decision; stub always confirms.
func Decide(in Input) Decision {
	return Decision{Decision: classifier.DecisionConfirm, Reason: "stub arbiter"}
}
