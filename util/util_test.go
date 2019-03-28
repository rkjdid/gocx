package util

import (
	"math"
	"testing"
)

func Test_helpers(t *testing.T) {
	if math.Abs(MidLog2F(2, 10))-4.47 > 0.1 {
		t.Error("MidLog2F", MidLog2F(2, 10))
	}
	if MidLog2(2, 10) != 4 {
		t.Errorf("MidLog2")
	}
	if MidLinearF(2, 10) != 6 {
		t.Errorf("MidLinearF")
	}
	if MidLinear(2, 10) != 6 {
		t.Errorf("MidLinear")
	}
	if MovOppositeF(1.95, 2, 3) != 2.05 {
		t.Errorf("MovOppositeF")
	}
	if MovOppositeF(0, 1, 4) != 2 {
		t.Errorf("MovOppositeF")
	}

	t.Log(RangeLinear(3, 0, 10))
	t.Log(RangeLinearF(3, 0, 10))
	t.Log(RangeLog2(3, 1, 10))
	t.Log(RangeLog2F(3, 1, 10))
}
