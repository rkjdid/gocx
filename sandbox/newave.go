package main

import (
	"fmt"
	"github.com/rkjdid/gocx"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/risk"
	"github.com/rkjdid/gocx/strategy"
	"gonum.org/v1/plot/vg"
	"log"
	"math"
	"time"
)

func Newave(x, bcur, qcur string, tf string, agg int, tf2 string, agg2 int, from, to time.Time) (*Result, error) {
	return NewWaveOpts{
		Exchange: x, Base: bcur, Quote: qcur, Tf1: tf, Agg1: agg, Tf2: tf2, Agg2: agg2, From: from, To: to,
	}.Backtest()
}

type NewWaveOpts struct {
	Exchange    string
	Base, Quote string
	From, To    time.Time

	Tf1, Tf2     string
	Agg1, Agg2   int
	Opts1, Opts2 strategy.MACDOpts
}

func (n NewWaveOpts) Backtest() (*Result, error) {
	hist1, err := LoadHistorical(n.Exchange, n.Base, n.Quote, n.Tf1, n.Agg1, n.From, n.To)
	if err != nil {
		return nil, err
	}
	hist2, err := LoadHistorical(n.Exchange, n.Base, n.Quote, n.Tf2, n.Agg2, n.From, n.To)
	if err != nil {
		return nil, err
	}

	// init chart
	if gocx.Chart {
		chart.SetTitles(fmt.Sprintf("%s:%s%s", n.Exchange, n.Base, n.Quote), "", "")
		chart.AddOHLCVs(hist2.Data)
	}

	// init & feed strategy
	macd1 := strategy.NewMACD(12, 26, 9)
	macd2 := strategy.NewMACD(12, 26, 9)
	if hist1.Timeframe > hist2.Timeframe {
		hist1, hist2 = hist2, hist1
		macd1, macd2 = macd2, macd1
	}

	var k0 = 1.0
	var k = k0
	var result = &Result{}
	var pos *risk.Position

	j := 0
	var sig1, sig2, last strategy.Signal
	for _, x := range hist1.Data {
		// Positions management ?
		// are we closing ?
		if pos != nil && pos.Active() {
			potentialNet := pos.NetOnClose(x.Close)

			// target +20%
			if potentialNet > 0 && potentialNet > 0.1*pos.Cost() {
				pos.PaperCloseAt(x.Close, x.Timestamp.T())
			}
			// stop   -5%
			if potentialNet < 0 && potentialNet < -0.025*pos.Cost() {
				pos.PaperCloseAt(x.Close, x.Timestamp.T())
			}

			if pos.State == risk.Closed {
				tradeCount.WithLabelValues("sell", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
				k += pos.Net()
			}
		}

		sig1 = macd1.AddTick(x)
		if len(hist2.Data) > j+1 && x.Timestamp.T().After(hist2.Data[j+1].Timestamp.T()) {
			j += 1
			sig2 = macd2.AddTick(hist2.Data[j])
		}

		var trigger strategy.Signal
		if sig1 != strategy.NoSignal && macd2.LastSignal.Action == sig1.Action {
			trigger = sig1
		} else if sig2 != strategy.NoSignal && macd1.LastSignal.Action == sig2.Action {
			trigger = sig2
		}

		// print signals individually
		if sig1 != strategy.NoSignal {
			//if gocx.Chart {
			//	chart.AddSignal(sig1.Time, sig1.Action == strategy.Buy, false, 10000)
			//}
			sigCount.WithLabelValues("fast", sig1.Action.String()).Inc()
		}
		if sig2 != strategy.NoSignal {
			//if gocx.Chart {
			//	chart.AddSignal(sig2.Time, sig2.Action == strategy.Buy, true, 20000)
			//}
			sigCount.WithLabelValues("slow", sig1.Action.String()).Inc()
		}

		tt := trigger.Action == strategy.Buy
		if trigger != strategy.NoSignal && trigger.Action != last.Action {
			if gocx.Chart {
				chart.AddSignal(trigger.Time, tt, true, 0)
			}
			sigCount.WithLabelValues("newave", trigger.Action.String()).Inc()
			last = trigger

			// buy signal -> open position
			if tt && (pos == nil || pos.State == risk.Closed) {
				pos = risk.NewPosition(x.Timestamp.T(), n.Base, n.Quote, risk.Long)
				pos.PaperBuyAt(k/x.Close, x.Close, x.Timestamp.T())
				tradeCount.WithLabelValues("buy", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
				result.Positions = append(result.Positions, pos)
			}
		}
	}
	result.Score = k/k0 - 1

	// draw chart
	if gocx.Chart {
		macd1.Draw()
		chart.NextLineTheme()
		macd2.Draw()
		cname := fmt.Sprintf("img/%sn.Exchange%s%s.png", *prefix, n.Base, n.Quote)
		width := vg.Length(math.Max(float64(len(hist1.Data)), 1200))
		height := width / 1.77

		err = chart.Save(width, height, false, cname)
		if err != nil {
			return result, err
		}
		log.Printf("saved \"%s\"", cname)
	}

	return result, nil
}
