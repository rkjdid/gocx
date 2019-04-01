package util

import (
	"math"
	"testing"
	"time"
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

	t.Log("RangeLinear(3, 0, 10)", RangeLinear(3, 0, 10))
	t.Log("RangeLinearF(3, 0, 10)", RangeLinearF(3, 0, 10))
	t.Log("RangeLog2(3, 1, 10)", RangeLog2(3, 1, 10))
	t.Log("RangeLog2F(3, 1, 10)", RangeLog2F(3, 1, 10))
}

func Test_UnixToTime(t *testing.T) {
	assertEq := func(i int64, d time.Time) {
		if res := UnixToTime(i); !res.Equal(d) {
			t.Errorf("unexpected UnixToTime(%d): %s != %s", i, res, d)
		}
	}

	v, exp := int64(1554111392), time.Date(2019, 4, 1, 9, 36, 32, 0, time.UTC)
	assertEq(v, exp)

	v, exp = int64(-1), time.Time{}
	assertEq(v, exp)

	v, exp = int64(1550000000000), time.Date(2019, 2, 12, 19, 33, 20, 0, time.UTC)
	assertEq(v, exp)

	v = v / 1000
	assertEq(v, exp)
}
