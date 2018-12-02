package backtest

import (
	"bytes"
	"fmt"
	"github.com/rkjdid/gocx/risk"
	"github.com/rkjdid/util"
	"time"
)

type Result struct {
	Score     float64
	Positions []*risk.Position
}

func (r *Result) UpdateScore() float64 {
	for _, p := range r.Positions {
		r.Score += p.Net()
	}
	return r.Score
}

func (r Result) ScorePerDay() float64 {
	sz := len(r.Positions)
	if r.Score == 0 && sz > 0 {
		r.UpdateScore()
	}
	for i := sz - 1; i >= 0; i-- {
		pos := r.Positions[i]
		if pos.CloseTime.Equal(time.Time{}) {
			continue
		}
		return r.Score / (pos.CloseTime.Sub(r.Positions[0].OpenTime).Hours() / 24)
	}
	return 0
}

func (r Result) String() string {
	var wins, loses int
	for _, p := range r.Positions {
		if p.Net() < 0 {
			loses++
		} else if p.Net() > 0 {
			wins++
		}
	}
	return fmt.Sprintf("net/day: %5.2f%%, +pos: %2d, -pos: %2d", r.ScorePerDay()*100, wins, loses)
}

func (r Result) JSON() ([]byte, error) {
	var b bytes.Buffer
	err := util.WriteJson(r, &b)
	return b.Bytes(), err
}
