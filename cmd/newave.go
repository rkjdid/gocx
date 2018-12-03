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

type NewaveConfig struct {
	backtest.CommonConfig

	TimeframeSlow      backtest.Timeframe
	MACDFast, MACDSlow strategy.MACDOpts
}

type NewaveResult struct {
	Config NewaveConfig
	backtest.Result
}

var (
	newaveCmd = TraverseRunHooks(&cobra.Command{
		Use:   "newave",
		Short: "Newave strategy original",
		Long: `Newave relies on MACD1(tf1) and MACD2(tf2), when both are green, buy asset

Usually MACD1 & MACD2 config are the same but they can differ..`,
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
				NewaveTop(n)
			} else if len(args) == 2 {
				NewaveOne(args[0], args[1])
			}
		},
	})
)

func init() {
	newaveCmd.LocalFlags().StringVar(&tf2, "tf2", "", tfFlagHelper())
}

func NewaveTop(n int) {
	tickers, err := binance.FetchTopTickers("", "BTC")
	if err != nil {
		log.Fatal(err)
	}
	if n <= 0 {
		n = len(tickers)
	}
	for _, v := range tickers[:n] {
		NewaveOne(v.Base, v.Quote)
	}
}

func NewaveOne(bcur, qcur string) {
	macd1 := strategy.MACDOpts{12, 16, 9}
	macd2 := macd1
	res, err := Newave(x, bcur, qcur, ttf, ttf2, macd1, macd2, tfrom, tto)
	if err != nil {
		log.Printf("Newave %s:%s%s - %s", x, bcur, qcur, err)
		return
	}
	for _, p := range res.Positions {
		fmt.Println(p)
	}
	fmt.Println(res)

	// save to redis
	err = db.Save(res)
	if err != nil {
		log.Println("db: error saving backtest result:", err)
	}
}

func Newave(x, bcur, qcur string,
	tf, tf2 backtest.Timeframe,
	macdFast, macdSlow strategy.MACDOpts,
	from, to time.Time) (
	*NewaveResult, error) {
	return NewaveConfig{
		CommonConfig: backtest.CommonConfig{
			Exchange: x,
			Base:     bcur, Quote: qcur, Timeframe: tf, From: from, To: to,
		},
		TimeframeSlow: tf2,
		MACDFast:      macdFast,
		MACDSlow:      macdSlow,
	}.Backtest()
}

func (n NewaveConfig) Backtest() (*NewaveResult, error) {
	if !n.Timeframe.IsValid() {
		return nil, fmt.Errorf("invalid tf: %s", n.Timeframe)
	}
	if !n.TimeframeSlow.IsValid() {
		return nil, fmt.Errorf("invalid tf2: %s", n.TimeframeSlow)
	}

	histFast, err := backtest.LoadHistorical(n.Exchange, n.Base, n.Quote, n.Timeframe, n.From, n.To)
	if err != nil {
		return nil, err
	}
	histSlow, err := backtest.LoadHistorical(n.Exchange, n.Base, n.Quote, n.TimeframeSlow, n.From, n.To)
	if err != nil {
		return nil, err
	}

	// set actual n.From after LoadHistorical
	n.From = histFast.From

	// init chart
	if chartFlag {
		chart.SetTitles(fmt.Sprintf("%s:%s%s", n.Exchange, n.Base, n.Quote), "", "")
		chart.AddOHLCVs(histSlow.Data)
	}

	// init & feed strategy
	macdFast := n.MACDFast.NewMACD()
	macdSlow := n.MACDSlow.NewMACD()
	if histFast.Timeframe.ToDuration() > histSlow.Timeframe.ToDuration() {
		histFast, histSlow = histSlow, histFast
		macdFast, macdSlow = macdSlow, macdFast
	}

	var k0 = 1.0
	var k = k0
	var result = NewaveResult{Config: n}
	var pos *risk.Position

	j := 0
	var sig1, sig2, last strategy.Signal
	for _, x := range histFast.Data {
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

		sig1 = macdFast.AddTick(x)
		if len(histSlow.Data) > j+1 && x.Timestamp.T().After(histSlow.Data[j+1].Timestamp.T()) {
			j += 1
			sig2 = macdSlow.AddTick(histSlow.Data[j])
		}

		var trigger strategy.Signal
		if sig1 != strategy.NoSignal && macdSlow.LastSignal.Action == sig1.Action {
			trigger = sig1
		} else if sig2 != strategy.NoSignal && macdFast.LastSignal.Action == sig2.Action {
			trigger = sig2
		}

		// print signals individually
		if sig1 != strategy.NoSignal {
			sigCount.WithLabelValues("fast", sig1.Action.String()).Inc()
		}
		if sig2 != strategy.NoSignal {
			sigCount.WithLabelValues("slow", sig1.Action.String()).Inc()
		}

		tt := trigger.Action == strategy.Buy
		if trigger != strategy.NoSignal && trigger.Action != last.Action {
			if chartFlag {
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
	result.UpdateScore()

	// draw chart
	if chartFlag {
		macdFast.Draw()
		chart.NextLineTheme()
		macdSlow.Draw()
		cname := fmt.Sprintf("img/newave_%s%s.png", n.Base, n.Quote)
		width := vg.Length(math.Max(float64(len(histFast.Data)), 1200))
		height := width / 1.77

		err = chart.Save(width, height, false, cname)
		if err != nil {
			return &result, err
		}
		log.Printf("saved \"%s\"", cname)
	}

	return &result, nil
}

// Digest is db.Digester implementation with json data and a id-hash.
func (r NewaveResult) Digest() (id string, data []byte, err error) {
	var b bytes.Buffer
	err = json.NewEncoder(&b).Encode(r)
	if err != nil {
		return
	}
	data = b.Bytes()
	shasum := sha256.Sum256(data)
	id = fmt.Sprintf("newave:%s%s:%x", r.Config.Base, r.Config.Quote, shasum[:10])
	return
}

func (n NewaveConfig) String() string {
	return fmt.Sprintf("%s:%8s - tf: %s - %s to %s - macd(%s, %s) & macd(%s, %s)",
		n.Exchange, fmt.Sprint(n.Base, n.Quote), n.Timeframe,
		n.From.Format("02/01/06"), n.To.Format("02/01/2006"),
		n.Timeframe, n.MACDFast, n.TimeframeSlow, n.MACDSlow,
	)
}

func (r NewaveResult) String() string {
	return fmt.Sprintf("%s - %s", r.Config, r.Result)
}
