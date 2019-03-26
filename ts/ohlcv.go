package ts

import (
	"github.com/ccxt/ccxt/go/util"
	"github.com/montanaflynn/stats"
	"gonum.org/v1/plot/plotter"
	"math"
	"time"
)

type OHLCV struct {
	Timestamp util.JSONTime `json:"time"`
	Open      float64       `json:"open"`
	High      float64       `json:"high"`
	Low       float64       `json:"low"`
	Close     float64       `json:"close"`
	Volume    float64       `json:"volumefrom"`
}

func (o OHLCV) Pivot() float64 {
	return (o.High + o.Low + o.Open) / 3
}

func (o OHLCV) IsZero() bool {
	return o.Volume == 0 && o.Open == 0 && o.Close == 0
}

func (o OHLCV) Sub(on OHLCV) time.Duration {
	return time.Time(o.Timestamp).Sub(time.Time(on.Timestamp))
}

func (o OHLCV) IsAlmost(o0 OHLCV) bool {
	return math.Abs(float64(o.Sub(o0))) <= float64(time.Minute*30)
}

func (o OHLCV) IsNextTo(o0 OHLCV, tf time.Duration) bool {
	return o.Sub(o0) >= time.Duration(float64(tf)*0.95) &&
		o.Sub(o0) <= time.Duration(float64(tf)*1.05)
}

func (o0 OHLCV) IsPrevOf(o OHLCV, tf time.Duration) bool {
	return o.Sub(o0) >= time.Duration(float64(tf)*0.95) &&
		o.Sub(o0) <= time.Duration(float64(tf)*1.05)
}

type OHLCVs []OHLCV

func (o OHLCVs) Range() (x0, xn, y0, yn float64) {
	return o.X0(), o.XN(), o.Y0(), o.YN()
}

func (o OHLCVs) X0() float64 {
	if len(o) >= 1 {
		return float64(o.X0T().Unix())
	}
	return 0
}

func (o OHLCVs) X0T() time.Time {
	return time.Time(time.Time(o[0].Timestamp))
}

func (o OHLCVs) XN() float64 {
	if len(o) >= 1 {
		return float64(o.X0T().Unix())
	}
	return 0
}

func (o OHLCVs) XNT() time.Time {
	return time.Time(time.Time(o[len(o)-1].Timestamp))
}

func (o OHLCVs) Y0() float64 {
	f, _ := stats.Min(stats.Float64Data(o.Low()))
	return f
}

func (o OHLCVs) YN() float64 {
	f, _ := stats.Max(stats.Float64Data(o.High()))
	return f
}

func (o OHLCVs) XStep() float64 {
	if len(o) >= 2 {
		return float64(o.XStepDuration().Seconds())
	}
	return 0
}

func (o OHLCVs) XStepDuration() time.Duration {
	return time.Time(o[1].Timestamp).Sub(time.Time(o[0].Timestamp))
}

func (o OHLCVs) ToXYerSlice(data ...[]float64) (ts []plotter.XYer) {
	for _, v := range data {
		ts = append(ts, o.ToXYer(v))
	}
	return ts
}

func (o OHLCVs) ToXYer(data []float64) (xyer plotter.XYer) {
	return TimeSeries{
		data, o.X0() + o.XStep()*float64(len(o)-len(data)), o.XStep(),
	}.CleanCopy()
}

// TrimLeft returns a slice with zero-elements removed from the beginning of o.
func (o OHLCVs) TrimLeft() OHLCVs {
	if o == nil {
		return o
	}
	var i int
	for i = 0; i < len(o) && o[i].IsZero(); i++ {
	}
	return o[i:]
}

// TrimLeft returns a slice with zero-elements removed from the end of o.
func (o OHLCVs) TrimRight() OHLCVs {
	if o == nil {
		return o
	}
	var i int
	for i = len(o) - 1; i >= 0 && o[i].IsZero(); i-- {
	}
	return o[:i+1]
}

func (o OHLCVs) Trim() OHLCVs {
	return o.TrimLeft().TrimRight()
}

func (o OHLCVs) Clean() OHLCVs {
	const brokenRatio = 1024
	for i, v := range o {
		for _, val := range []*float64{&v.Open, &v.High, &v.Low, &v.Close} {
			if *val < 0 {
				continue
			}
			if *val == 0 ||
				(*val > brokenRatio*v.Open && *val > brokenRatio*v.Close) ||
				(*val > brokenRatio*v.Low && *val > brokenRatio*v.High) {
				if *val == v.Close {
					*val = v.Open
				} else {
					*val = v.Close
				}
				o[i] = v
			}
		}
	}
	return o
}

func (o OHLCVs) Open() (val []float64) {
	val = make([]float64, len(o))
	for i, v := range o {
		val[i] = v.Open
	}
	return val
}

func (o OHLCVs) High() (val []float64) {
	val = make([]float64, len(o))
	for i, v := range o {
		val[i] = v.High
	}
	return val
}

func (o OHLCVs) Low() (val []float64) {
	val = make([]float64, len(o))
	for i, v := range o {
		val[i] = v.Low
	}
	return val
}

func (o OHLCVs) Close() (val []float64) {
	val = make([]float64, len(o))
	for i, v := range o {
		val[i] = v.Close
	}
	return val
}

func (o OHLCVs) Pivot() (val []float64) {
	val = make([]float64, len(o))
	for i, v := range o {
		val[i] = v.Pivot()
	}
	return val
}

func (o OHLCVs) Volume() (val []float64) {
	val = make([]float64, len(o))
	for i, v := range o {
		val[i] = v.Volume
	}
	return val
}

func (o OHLCVs) Len() int {
	return len(o)
}

func (o OHLCVs) TOHLCV(i int) (t float64, op float64, h float64, l float64, c float64, v float64) {
	if i < 0 || i >= len(o) {
		return
	}
	return float64(time.Time(o[i].Timestamp).Unix()), o[i].Open, o[i].High, o[i].Low, o[i].Close, o[i].Volume
}

// TrimLeft returns a slice of data that exclude leading 0s,
// and the amount of elements excluded, no copy is made.
func TrimLeft(data []float64) (out []float64, n int) {
	for n = 0; n < len(data) && data[n] == 0; n++ {
	}
	return data[n:], n
}

// TrimRight returns a slice of data that exclude trailing 0s,
// and the amount of elements excluded, no copy is made.
func TrimRight(data []float64) (out []float64, n int) {
	for n = len(data) - 1; n >= 0 && data[n] == 0; n-- {
	}
	return data[:n+1], len(data) - n - 1
}

// Trim returns a slice of data that exclude leading and trailing 0s,
// and the amount of elements excluded respectively
// from the beginning and end of slice. No copy is made.
func Trim(data []float64) (out []float64, i, j int) {
	out, i = TrimLeft(data)
	out, j = TrimRight(out)
	return out, i, j
}

// TimeSeries represent a set of data points mapped to a X0 + n*XStep value.
// Any pair of points (Data[n], Data[n+1]) has a distance of XStep on the x axis.
type TimeSeries struct {
	Data  []float64
	X0    float64
	XStep float64
}

// CleanCopy returns a copy of ts with 0s trimmed from beginning and end,
// X0 shifted accordingly, and any other 0 value replaced with the average
// between its left and right neighbour.
func (ts TimeSeries) CleanCopy() TimeSeries {
	var ts2 TimeSeries
	var i int
	ts2.Data, i, _, _ = CleanFloats(ts.Data, Average)
	ts2.XStep = ts.XStep
	ts2.X0 = ts.X0 + float64(i)*ts.XStep
	return ts2
}

func (ts TimeSeries) Len() int {
	return len(ts.Data)
}

func (ts TimeSeries) XY(i int) (float64, float64) {
	if i < 0 || i >= len(ts.Data) {
		return 0, 0
	}
	return float64(i)*ts.XStep + ts.X0, ts.Data[i]
}

// CleanFloats returns a copy of data without any 0 value.
// First we remove leading and trailign 0s with Trim(), then we
// copy the data and replace any in-between 0 value using fn.
func CleanFloats(data []float64, fn func(left, right float64) float64,
) (out []float64, i, j, k int) {
	var tf []float64
	tf, i, j = Trim(data)
	out = make([]float64, len(tf))
	copy(out, tf)
	if len(out) <= 1 {
		if len(out) == 1 && math.IsNaN(out[0]) {
			out[0] = 0
			k = 1
		}
		return out, i, j, k
	}

	var x int
	var isZero = func(f float64) bool {
		return f == 0 || math.IsNaN(f)
	}
	// first point
	if isZero(out[x]) {
		k++
		out[x] = out[x+1]
	}
	// middle values
	for x = 1; x < len(out)-1; x++ {
		if isZero(out[x]) {
			k++
			out[x] = fn(out[x-1], out[x+1])
		}
	}
	// last point
	if isZero(out[x]) {
		k++
		out[x] = out[x-1]
	}
	return out, i, j, k
}

func PickLeft(left, right float64) float64 { return left }

func PickRight(left, right float64) float64 { return right }

func Average(left, right float64) float64 { return (left + right) / 2.0 }
