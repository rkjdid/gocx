package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/prometheus/client_golang/prometheus"
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

var (
	newave2Cmd = &cobra.Command{
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
				BacktestTopNewave2(n)
			} else if len(args) == 2 {
				BacktestOneNewave2(args[0], args[1])
			}
		},
	}
)

func init() {
	backtestCmd.AddCommand(newave2Cmd)
}

func BacktestTopNewave2(n int) {
	tickers, err := binance.FetchTopTickers("", "BTC")
	if err != nil {
		log.Fatal(err)
	}
	if n <= 0 {
		n = len(tickers)
	}
	for _, v := range tickers[:n] {
		BacktestOneNewave2(v.Base, v.Quote)
	}
}

func BacktestOneNewave2(bcur, qcur string) {
	ratio := 4
	macd1 := strategy.MACDOpts{12, 16, 9}
	macd2 := strategy.MACDOpts{ratio * 12, ratio * 26, 9}
	res, err := Newave(x, bcur, qcur, tf, agg, macd1, macd2, tfrom, tto)
	if err != nil {
		log.Printf("Newave %s:%s%s - %s", x, bcur, qcur, err)
		return
	}
	for _, p := range res.Positions {
		fmt.Println(p)
	}
	fmt.Println(res)

	// save to redis
	err = res.Save(db)
	if err != nil {
		log.Println("redis: error saving backtest result:", err)
	}
}

type NewWaveResult struct {
	Config NewWaveOpts
	backtest.Result
}

type NewWaveOpts struct {
	Exchange    string
	Base, Quote string
	From, To    time.Time
	Tf          string
	Agg         int

	MACDFast, MACDSlow strategy.MACDOpts
}

var (
	sigCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signal", Help: "various signals",
	}, []string{"name", "action"})

	tradeCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trade", Help: "trades",
	}, []string{"direction", "quantity", "price"})
)

func Newave(x, bcur, qcur string, tf string, agg int, fastOpts, slowOpts strategy.MACDOpts, from, to time.Time) (
	*NewWaveResult, error) {
	return NewWaveOpts{
		Exchange: x, Base: bcur, Quote: qcur, Tf: tf, Agg: agg, From: from, To: to,
		MACDFast: fastOpts, MACDSlow: slowOpts,
	}.Backtest()
}

func (n NewWaveOpts) Backtest() (*NewWaveResult, error) {
	hist, err := backtest.LoadHistorical(n.Exchange, n.Base, n.Quote, n.Tf, n.Agg, n.From, n.To)
	if err != nil {
		return nil, err
	}

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
	var result = &NewWaveResult{Config: n}
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
	if chartFlag {
		macdFast.Draw()
		chart.NextLineTheme()
		macdSlow.Draw()
		cname := fmt.Sprintf("img/n.Exchange%s%s.png", n.Base, n.Quote)
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

// DB / Data
func (n NewWaveOpts) String() string {
	return fmt.Sprintf("%s:%8s - tf: %d %s - %s to %s - macd1%s macd2%s",
		n.Exchange, fmt.Sprint(n.Base, n.Quote), n.Agg, n.Tf,
		n.From.Format("02/01/06"), n.To.Format("02/01/2006"),
		n.MACDFast, n.MACDSlow,
	)
}

func (r NewWaveResult) String() string {
	return fmt.Sprintf("%s - %s", r.Config, r.Result)
}

func (r NewWaveResult) JSON() ([]byte, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(r)
	return b.Bytes(), err
}

func (r NewWaveResult) Digest() (id string, data []byte, err error) {
	data, err = r.JSON()
	if err != nil {
		return
	}
	shasum := sha256.Sum256(data)
	id = fmt.Sprintf("newave:%s%s:%x", r.Config.Base, r.Config.Quote, shasum[:10])
	return
}

func (r NewWaveResult) Save(db *redis.Client) (err error) {
	id, data, err := r.Digest()
	if err != nil {
		return err
	}
	err = db.Cmd("MULTI").Err
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = db.Cmd("DISCARD")
		} else {
			err = db.Cmd("EXEC").Err
		}
	}()
	err = db.Cmd("SET", id, data).Err
	if err != nil {
		return err
	}
	return db.Cmd("ZADD", "results", r.ScorePerDay(), id).Err
}

func LoadJSON(db *redis.Client, id string, v interface{}) error {
	resp := db.Cmd("GET", id)
	if resp.Err != nil {
		return resp.Err
	}
	b, err := resp.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}