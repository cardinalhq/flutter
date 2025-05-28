package traceproducer

import (
	"math/rand/v2"
	"testing"
)

func TestScaledKindaNormal_Range(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 54))
	for range 1000 {
		val := scaledKindaNormal(r)
		if val < 0 || val > 1 {
			t.Errorf("scaledKindaNormal returned value out of range: %v", val)
		}
	}
}
