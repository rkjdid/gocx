package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/risk"
	"github.com/rkjdid/gocx/scraper/binance"
	"github.com/rkjdid/gocx/strategy"
	"github.com/spf13/cobra"
	"gonum.org/v1/plot/vg"
	"log"
	"math"
	"time"
)

type Newave2Config struct {
	backtest.CommonConfig

	MACDFast, MACDSlow strategy.MACDOpts
}

type Newave2Result struct {
	Config Newave2Config
	backtest.Result
}

var (
	newave2Cmd = TraverseRunHooks(&cobra.Command{
		Use:   "newave2",
		Short: "Newave strategy v2",
		Long:  `Newave2 relies on 2 MACD for the same timeframe, when both are green, buy asset`,
		Args: func(cmd *cobra.Command, args []string) error {
			switch len(args) {
			case 0:
				return nil
			case 2:
				return nil
			default:
				return fmt.Errorf("expected no args or <base> <quote>")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Newave2Top(n)
			} else if len(args) == 2 {
				Newave2One(args[0], args[1])
			}
		},
	})
)

func Newave2Top(n int) {
	tickers, err := binance.FetchTopTickers("", "BTC")
	if err != nil {
		log.Fatal(err)
	}
	if n <= 0 {
		n = len(tickers)
	}
	for _, v := range tickers[:n] {
		Newave2One(v.Base, v.Quote)
	}
}

func Newave2One(bcur, qcur string) {
	ratio := 4
	macd1 := strategy.MACDOpts{12, 16, 9}
	macd2 := strategy.MACDOpts{ratio * 12, ratio * 26, 9}
	res, err := Newave2(x, bcur, qcur, ttf, macd1, macd2, tfrom, tto)
	if err != nil {
		log.Printf("Newave2 %s:%s%s - %s", x, bcur, qcur, err)
		return
	}
	for _, p := range res.Positions {
		fmt.Println(p)
	}
	fmt.Println(res)

	// save to redis
	id, err := db.Save(res)
	if err != nil {
		log.Println("db: error saving backtest result:", err)
	}
	err = db.ZADD("results", id, res.ZScore())
	if err != nil {
		log.Println("db: error zadding backtest result:", err)
	}
}

func Newave2(x, bcur, qcur string, tf backtest.Timeframe, fastOpts, slowOpts strategy.MACDOpts, from, to time.Time) (
	*Newave2Result, error) {
	return Newave2Config{
		CommonConfig: backtest.CommonConfig{
			Exchange: x, Base: bcur, Quote: qcur, Timeframe: tf, From: from, To: to},
		MACDFast: fastOpts, MACDSlow: slowOpts,
	}.Backtest()
}

func (n Newave2Config) Backtest() (*Newave2Result, error) {
	if !n.Timeframe.IsValid() {
		return nil, fmt.Errorf("invalid tf: %s", n.Timeframe)
	}
	hist, err := backtest.LoadHistorical(n.Exchange, n.Base, n.Quote, n.Timeframe, n.From, n.To)
	if err != nil {
		return nil, err
	}
	// set actual n.From after LoadHistorical
	n.From = hist.From

	// init chart
	if chartFlag {
		chart.SetTitles(fmt.Sprintf("%s:%s%s", n.Exchange, n.Base, n.Quote), "", "")
		chart.AddOHLCVs(hist.Data)
	}

	// init & feed strategy
	macdFast := n.MACDFast.NewMACD()
	macdSlow := n.MACDSlow.NewMACD()
	// todo swap fast <-> slow if needed

	var k0 = 1.0
	var k = k0
	var result = &Newave2Result{Config: n}
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
			sigCount.WithLabelValues("fast", sigFast.Action.String()).Inc()
		}
		if sigSlow != strategy.NoSignal {
			sigCount.WithLabelValues("slow", sigFast.Action.String()).Inc()
		}

		tt := trigger.Action == strategy.Buy
		if trigger != strategy.NoSignal && trigger.Action != last.Action {
			if chartFlag {
				chart.AddSignal(trigger.Time, tt, true, 0)
			}
			sigCount.WithLabelValues("newave2", trigger.Action.String()).Inc()
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

	// draw chart
	if chartFlag {
		macdFast.Draw()
		chart.NextLineTheme()
		macdSlow.Draw()
		cname := fmt.Sprintf("img/newave2_%s%s.png", n.Base, n.Quote)
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

// Digest is db.Digester implementation with json data and a id-hash.
func (r Newave2Result) Digest() (id string, data []byte, err error) {
	var b bytes.Buffer
	err = json.NewEncoder(&b).Encode(r)
	if err != nil {
		return
	}
	data = b.Bytes()
	shasum := sha256.Sum256(data)
	id = fmt.Sprintf("newave2:%s%s:%x", r.Config.Base, r.Config.Quote, shasum[:10])
	return
}

func (n Newave2Config) String() string {
	return fmt.Sprintf("newave2 %8s - tf: %s - %s to %s - macd1(%s) macd2(%s)",
		fmt.Sprint(n.Base, n.Quote), n.Timeframe,
		n.From.Format("02/01/06"), n.To.Format("02/01/2006"),
		n.MACDFast, n.MACDSlow,
	)
}

func (r Newave2Result) String() string {
	return fmt.Sprintf("%s - %s", r.Config, r.Result)
}
