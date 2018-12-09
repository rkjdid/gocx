package cmd

import (
	"math"
	"testing"
)

func Test_helpers(t *testing.T) {
	if math.Abs(midLog2F(2, 10))-4.47 > 0.1 {
		t.Error("midLog2F", midLog2F(2, 10))
	}
	if midLog2(2, 10) != 4 {
		t.Errorf("midLog2")
	}
	if midLinearF(2, 10) != 6 {
		t.Errorf("midLinearF")
	}
	if midLinear(2, 10) != 6 {
		t.Errorf("midLinear")
	}
	if movOppositeF(1.95, 2) != 2.05 {
		t.Errorf("movOppositeF")
	}
	if movOppositeF(1, 2) != 3 {
		t.Errorf("movOppositeF")
	}
}
