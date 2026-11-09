package engine

import "time"

// SelectionTrace is an observational snapshot; scores are the exact values used
// by AcquireForHint. Nil scores mean the adapter did not enter the final contest.
type SelectionTrace struct {
	Time       time.Time            `json:"time"`
	Family     string               `json:"family"`
	Automatic  bool                 `json:"automatic"`
	Policy     string               `json:"policy"`
	Outcome    string               `json:"outcome"`
	Selected   string               `json:"selected,omitempty"`
	Candidates []SelectionCandidate `json:"candidates"`
}
type SelectionCandidate struct {
	Name       string   `json:"name"`
	State      string   `json:"state"`
	Weight     float64  `json:"weight"`
	Active     int64    `json:"active"`
	Load       int64    `json:"load"`
	LatencyMs  int64    `json:"latencyMs"`
	CooldownMs int64    `json:"cooldownMs,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Reason     string   `json:"reason"`
}
