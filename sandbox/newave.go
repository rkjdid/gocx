package main

import (
	"fmt"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/risk"
	"github.com/rkjdid/gocx/strategy"
	"gonum.org/v1/plot/vg"
	"log"
	"math"
	"time"
)

func Newave(x, bcur, qcur string, tf string, agg int, tf2 string, agg2 int, from, to time.Time) error {
	hist1, err := LoadHistorical(x, bcur, qcur, tf, agg, from, to)
	if err != nil {
		return err
	}
	hist2, err := LoadHistorical(x, bcur, qcur, tf2, agg2, from, to)
	if err != nil {
		return err
	}

	// init chart
	chart.SetTitles(fmt.Sprintf("%s:%s%s", x, bcur, qcur), "", "")
	chart.AddOHLCVs(hist2.Data)

	// init & feed strategy
	macd1 := strategy.NewMACD(12, 26, 9)
	macd2 := strategy.NewMACD(12, 26, 9)
	if hist1.Timeframe > hist2.Timeframe {
		hist1, hist2 = hist2, hist1
		macd1, macd2 = macd2, macd1
	}

	var k0 = 1000.0
	var k = k0
	var positions []*risk.Position
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
		t1 := sig1.Action == strategy.Buy
		if sig1 != strategy.NoSignal {
			_ = t1
			//chart.AddSignal(sig1.Time, t1, false, 10000)
			sigCount.WithLabelValues("fast", sig1.Action.String()).Inc()
		}
		t2 := sig2.Action == strategy.Buy
		if sig2 != strategy.NoSignal {
			_ = t2
			//chart.AddSignal(sig2.Time, t2, true, 20000)
			sigCount.WithLabelValues("slow", sig1.Action.String()).Inc()
		}

		tt := trigger.Action == strategy.Buy
		if trigger != strategy.NoSignal && trigger.Action != last.Action {
			//chart.AddSignal(trigger.Time, tt, true, 0)
			// todo print signals using a prometheus Collector via sigCount var
			sigCount.WithLabelValues("newave", trigger.Action.String()).Inc()
			last = trigger

			// buy signal -> open position
			if tt && (pos == nil || pos.State == risk.Closed) {
				pos = risk.NewPosition(x.Timestamp.T(), bcur, qcur, risk.Long)
				pos.PaperBuyAt(k/x.Close, x.Close, x.Timestamp.T())
				tradeCount.WithLabelValues("buy", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
				positions = append(positions, pos)
			}
		}
	}

	for _, p := range positions {
		fmt.Println(p)
	}
	fmt.Println()
	fmt.Printf("initial: %f     net: %.2f    work: %.2f%%\n", k0, k, 100*(k/k0-1))

	// draw chart
	macd1.Draw()
	chart.NextLineTheme()
	macd2.Draw()
	cname := fmt.Sprintf("img/%sx%s%s.png", *prefix, bcur, qcur)
	width := vg.Length(math.Max(float64(len(hist1.Data)), 1200))
	height := width / 1.77

	err = chart.Save(width, height, false, cname)
	if err != nil {
		return err
	}
	log.Printf("saved \"%s\"", cname)
	return nil
}
