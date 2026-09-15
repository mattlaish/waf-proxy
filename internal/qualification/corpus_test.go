package qualification

import "testing"

func TestZeroFalseNegativeGate(t *testing.T) {
	r := Compare(Sample{ID:"x"}, []string{"942100"}, []string{"942100"})
	if !ZeroFalseNegative([]DifferentialResult{r}) { t.Fatal("expected pass") }
}
