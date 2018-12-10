package backtest

import (
	"fmt"
	"github.com/rkjdid/gocx/trading"
	"time"
)

type Result struct {
	Positions []*trading.Position
	From, To  time.Time
	Score     float64
	Z         float64
}

// ZScore implements db.ZScorer. It returns r.Score / day.
func (r *Result) ZScore() float64 {
	// update score if needed
	sz := len(r.Positions)
	if r.Score == 0 && sz > 0 {
		r.UpdateScore()
	}
	return r.Z
}

func (r *Result) UpdateScore() {
	r.Score = 0
	for _, p := range r.Positions {
		r.Score += p.Net()
	}
	nbDays := r.To.Sub(r.From).Hours() / 24
	r.Z = r.Score / nbDays
	return
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
	return fmt.Sprintf("zscore: %5.3f, total: %.1f%%, +pos: %2d, -pos: %2d", 100*r.ZScore(), 100*r.Score, wins, loses)
}
