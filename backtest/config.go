package backtest

import (
	"time"
)

type Common struct {
	Exchange    string
	Base, Quote string
	From, To    time.Time
	Timeframe   Timeframe
	RiskProfile
}

type RiskProfile struct {
	TakeProfit float64
	StopLoss   float64
}
