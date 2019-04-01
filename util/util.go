package util

import (
	"math"
	"math/rand"
	"time"
)

func MovOpposite(v int, min, max int) int {
	return int(MovOppositeF(float64(v), float64(min), float64(max)))
}

func MovOppositeF(v float64, min, max float64) float64 {
	var bond float64
	if v > max {
		bond = max
	} else if v < min {
		bond = min
	} else {
		return v
	}
	return 2*bond - v
}

func MidLinear(min, max int) int {
	return RangeLinear(1, min, max)[0]
}

func MidLinear_(_, min, max int) int {
	return MidLinear(min, max)
}

func MidLinearF(min, max float64) float64 {
	return RangeLinearF(1, min, max)[0]
}

func MidLinearF_(_, min, max float64) float64 {
	return MidLinearF(min, max)
}

func MidLog2(min, max int) int {
	return RangeLog2(1, min, max)[0]
}

func MidLog2_(_, min, max int) int {
	return MidLog2(min, max)
}

func MidLog2F(min, max float64) float64 {
	return RangeLog2F(1, min, max)[0]
}

func MidLog2F_(_, min, max float64) float64 {
	return MidLog2F(min, max)
}

func RangeLinear(n, min, max int) []int {
	out := make([]int, n)
	step := float64(max-min) / float64(n+1)
	for i := 0; i < n; i++ {
		out[i] = int(math.Round(float64(min) + step*float64(i+1)))
	}
	return out
}

func RangeLinearF(n int, min, max float64) []float64 {
	if n <= 0 {
		return nil
	}
	out := make([]float64, n)
	step := (max - min) / (float64(n) + 1)
	for i := 0; i < n; i++ {
		out[i] = min + step*(float64(i)+1)
	}
	return out
}

func RangeLog2(n, min, max int) []int {
	out := make([]int, n)
	logMin := math.Log2(float64(min))
	logMax := math.Log2(float64(max))
	step := (logMax - logMin) / float64(n+1)
	for i := 0; i < n; i++ {
		out[i] = int(math.Round(math.Pow(2, logMin+step*(float64(i)+1))))
	}
	return out
}

func RangeLog2F(n int, min, max float64) []float64 {
	out := make([]float64, n)
	logMin := math.Log2(float64(min))
	logMax := math.Log2(float64(max))
	step := (logMax - logMin) / float64(n+1)
	for i := 0; i < n; i++ {
		out[i] = math.Pow(2, logMin+step*(float64(i)+1))
	}
	return out
}

func RandRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func RandRangeF(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func FixRangeFnF(v *float64, min, max float64, fn func(float64, float64, float64) float64) {
	if *v < min || *v > max {
		*v = fn(*v, min, max)
	}
}

func FixRangeFn(v *int, min, max int, fn func(int, int, int) int) {
	if *v < min || *v > max {
		*v = fn(*v, min, max)
	}
}

func FixRangeLinearF(v *float64, min, max float64) {
	FixRangeFnF(v, min, max, MidLinearF_)
}

func FixRangeLinear(v *int, min, max int) {
	FixRangeFn(v, min, max, MidLinear_)
}

func FixRangeLog2F(v *float64, min, max float64) {
	FixRangeFnF(v, min, max, MidLog2F_)
}

func FixRangeLog2(v *int, min, max int) {
	FixRangeFn(v, min, max, MidLog2_)
}

func UnixToTime(q int64) time.Time {
	if q <= 0 {
		return time.Time{}
	}
	if q > 1e11 {
		// 1e11 is year 5138, we assume we're dealing with millisecs
		return time.Unix(q/1000, (q%1000)*1e6)
	} else {
		// seconds timestamp
		return time.Unix(q, 0)
	}
}
