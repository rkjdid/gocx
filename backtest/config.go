package backtest

import (
	"time"
)

type CommonConfig struct {
	Exchange    string
	Base, Quote string
	From, To    time.Time
	Timeframe   Timeframe
}
