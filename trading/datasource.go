package trading

import (
	"github.com/rkjdid/gocx/ts"
	"time"
)

type DataSource interface {
	Feed() <-chan Tick
	Bondaries() (from, to time.Time)
}

type Tick struct {
	Timeframe ts.Timeframe
	ts.OHLCV
}
