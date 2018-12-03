package backtest

import (
	"fmt"
	"github.com/rkjdid/gocx/risk"
	"time"
)

type Result struct {
	Positions []*risk.Position
	Score     float64
}

// ZScore implements db.ZScorer. It returns r.Score / day.
func (r Result) ZScore() float64 {
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

func (r *Result) UpdateScore() float64 {
	for _, p := range r.Positions {
		r.Score += p.Net()
	}
	return r.Score
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
	return fmt.Sprintf("net/day: %5.2f%%, +pos: %2d, -pos: %2d", r.ZScore()*100, wins, loses)
}
