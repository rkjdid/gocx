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

func Newave(x, bcur, qcur string, tf string, agg int, slowOpts, fastOpts strategy.MACDOpts, from, to time.Time) (*Result, error) {
	return NewWaveOpts{
		Exchange: x, Base: bcur, Quote: qcur, Tf: tf, Agg: agg, From: from, To: to,
		MACDSlow: slowOpts, MACDFast: fastOpts,
	}.Backtest()
}

type NewWaveOpts struct {
	Exchange    string
	Base, Quote string
	From, To    time.Time
	Tf          string
	Agg         int

	MACDSlow, MACDFast strategy.MACDOpts
}

func (n NewWaveOpts) Backtest() (*Result, error) {
	hist, err := LoadHistorical(n.Exchange, n.Base, n.Quote, n.Tf, n.Agg, n.From, n.To)
	if err != nil {
		return nil, err
	}

	// init chart
	if gocx.Chart {
		chart.SetTitles(fmt.Sprintf("%s:%s%s", n.Exchange, n.Base, n.Quote), "", "")
		chart.AddOHLCVs(hist.Data)
	}

	// TODO use only 1 TF and use factor of TF

	// init & feed strategy
	macdFast := n.MACDFast.NewMACD()
	macdSlow := n.MACDSlow.NewMACD()
	//if hist.Timeframe > hist2.Timeframe {
	//	hist, hist2 = hist2, hist
	//	macdFast, macdSlow = macdSlow, macdFast
	//}

	var k0 = 1.0
	var k = k0
	var result = &Result{}
	var pos *risk.Position

	var sigFast, sigSlow, last strategy.Signal
	for _, x := range hist.Data {
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

		sigFast = macdFast.AddTick(x)
		sigSlow = macdSlow.AddTick(x)

		var trigger strategy.Signal
		if sigFast != strategy.NoSignal && macdSlow.LastSignal.Action == sigFast.Action {
			trigger = sigFast
		} else if sigSlow != strategy.NoSignal && macdFast.LastSignal.Action == sigSlow.Action {
			trigger = sigSlow
		}

		// print signals individually
		if sigFast != strategy.NoSignal {
			//if gocx.Chart {
			//	chart.AddSignal(sigFast.Time, sigFast.Action == strategy.Buy, false, 10000)
			//}
			sigCount.WithLabelValues("fast", sigFast.Action.String()).Inc()
		}
		if sigSlow != strategy.NoSignal {
			//if gocx.Chart {
			//	chart.AddSignal(sigSlow.Time, sigSlow.Action == strategy.Buy, true, 20000)
			//}
			sigCount.WithLabelValues("slow", sigFast.Action.String()).Inc()
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
		macdFast.Draw()
		chart.NextLineTheme()
		macdSlow.Draw()
		cname := fmt.Sprintf("img/%sn.Exchange%s%s.png", *prefix, n.Base, n.Quote)
		width := vg.Length(math.Max(float64(len(hist.Data)), 1200))
		height := width / 1.77

		err = chart.Save(width, height, false, cname)
		if err != nil {
			return result, err
		}
		log.Printf("saved \"%s\"", cname)
	}

	return result, nil
}
