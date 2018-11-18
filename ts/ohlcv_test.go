package ts

import (
	"github.com/ccxt/ccxt/go/util"
	"testing"
	"time"
)

func TestOHLCV_IsZero(t *testing.T) {
	var o OHLCV
	if !o.IsZero() {
		t.Errorf("zero-value struct should be zero")
	}
	o.Volume = 1
	if o.IsZero() {
		t.Errorf("volume != 0 shouldn't be zero")
	}
}

func TestOHLCV_IsNextTo(t *testing.T) {
	t0 := time.Now()
	c0 := OHLCV{
		Timestamp: util.JSONTime(t0.Add(-time.Minute * 59)),
	}
	cn := OHLCV{
		Timestamp: util.JSONTime(t0),
	}
	if !cn.IsNextTo(c0, time.Hour) {
		t.Errorf("! %s NextTo %s", cn.Timestamp, c0.Timestamp)
	}
	if !c0.IsPrevOf(cn, time.Hour) {
		t.Errorf("! %s PrevOf %s", c0.Timestamp, cn.Timestamp)
	}

	cz := OHLCV{
		Timestamp: util.JSONTime(t0.Add(-time.Hour * 2)),
	}
	if cz.IsPrevOf(cn, time.Hour) {
		t.Errorf("%s PrevOf %s", cz.Timestamp, cn.Timestamp)
	}
	if cz.IsNextTo(cn, time.Hour) {
		t.Errorf("%s NextTo %s", cz.Timestamp, cn.Timestamp)
	}
}

func TestOHLCVs_TrimLeft(t *testing.T) {
	data := make(OHLCVs, 10)
	if result := data.TrimLeft(); len(result) != 0 {
		t.Errorf("unexpected len(data): %d", len(result))
	}
	data[0] = OHLCV{Volume: 1}
	if result := data.TrimLeft(); len(result) != 10 {
		t.Errorf("data should be unmodified")
	}
	data[0] = OHLCV{}
	data[9] = OHLCV{Open: 1}
	if result := data.TrimLeft(); len(result) != 1 {
		t.Errorf("result should hold 1 elem, got %d", len(result))
	}
}

func TestOHLCVs_TrimRight(t *testing.T) {
	data := make(OHLCVs, 10)
	if result := data.TrimRight(); len(result) != 0 {
		t.Errorf("unexpected len(data): %d", len(result))
	}
	data[9] = OHLCV{Volume: 1}
	if result := data.TrimRight(); len(result) != 10 {
		t.Errorf("data should be unmodified")
	}
	data[9] = OHLCV{}
	data[0] = OHLCV{Open: 1}
	if result := data.TrimRight(); len(result) != 1 {
		t.Errorf("result should hold 1 elem, got %d", len(result))
	}
}

func TestTrimFloats(t *testing.T) {
	floatsEq := func(a []float64, b []float64) {
		if len(a) != len(b) {
			t.Errorf("expected %v got %v", a, b)
			return
		}
		for i := range a {
			if a[i] != b[i] {
				t.Errorf("at index %d: expected %f got %f", i, a[i], b[i])
			}
		}
	}
	f0 := []float64{0, 1, 2, 0}
	result, i := TrimLeft(f0)
	floatsEq(result, []float64{1, 2, 0})
	if i != 1 {
		t.Errorf("i != 1: %d", i)
	}
	result, i = TrimRight(f0)
	floatsEq(result, []float64{0, 1, 2})
	if i != 1 {
		t.Errorf("i != 1: %d", i)
	}
	result, i, j := Trim(f0)
	floatsEq(result, []float64{1, 2})
	if i != 1 {
		t.Errorf("i != 1: %d", i)
	}
	if j != 1 {
		t.Errorf("j != 1: %d", j)
	}

	for _, emptyFloats := range [][]float64{
		{}, {0}, {0, 0},
	} {
		l, _ := TrimLeft(emptyFloats)
		r, _ := TrimRight(emptyFloats)
		lr, _, _ := Trim(emptyFloats)

		floatsEq(l, []float64{})
		floatsEq(r, []float64{})
		floatsEq(lr, []float64{})
	}
}

func TestTimeSeries_Clean(t *testing.T) {
	ts := TimeSeries{
		Data:  []float64{0.0, 0.0, 1, 0.0, 2, 0.0, 0.0},
		X0:    10,
		XStep: 1,
	}
	expected := TimeSeries{
		Data:  []float64{1, 1.5, 2},
		X0:    12,
		XStep: 1,
	}
	result := ts.CleanCopy()
	if expected.X0 != result.X0 {
		t.Errorf("X0 values differ: expected %f, got %f", expected.X0, result.X0)
	}
	if len(expected.Data) != len(result.Data) {
		t.Errorf("len differ: expected %d, got %d", len(result.Data), len(expected.Data))
	}
	for i := range result.Data {
		if x, xx := expected.Data[i], result.Data[i]; x != xx {
			t.Errorf("item at index %d differ: expected %f, got %f", i, x, xx)
		}
	}
}
