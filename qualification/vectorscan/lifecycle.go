package vectorscan

const (
	Learning    = "LEARNING"
	Validated   = "VALIDATED"
	Accelerated = "ACCELERATED"
)

type LifecycleTransition struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
}

func ValidTransition(from, to string) bool {
	return (from == "" && to == Learning) ||
		(from == Learning && to == Validated) ||
		(from == Validated && to == Accelerated)
}
