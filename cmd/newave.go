package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/rkjdid/gocx/chart"
	_db "github.com/rkjdid/gocx/db"
	"github.com/rkjdid/gocx/scraper/binance"
	"github.com/rkjdid/gocx/trading"
	"github.com/rkjdid/gocx/trading/strategy"
	"github.com/spf13/cobra"
	"gonum.org/v1/plot/vg"
	"log"
	"math"
	"time"
)

const NewavePrefix = "newave"

type NewaveConfig struct {
	backtest.Source
	trading.Profile

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

func NewaveOne(bcur, qcur string) (*NewaveResult, error) {
	macd := strategy.MACDOpts{12, 16, 9}
	res, err := Newave(x, bcur, qcur, ttf, ttf2, macd, macd, tp, sl, tfrom, tto).Backtest()
	if err != nil {
		log.Printf("Newave %s:%s%s - %s", x, bcur, qcur, err)
		return nil, err
	}
	for _, p := range res.Positions {
		fmt.Println(p)
	}
	fmt.Println(res)

	// save to redis
	_, err = db.SaveZScorer(res, zkey)
	if err != nil {
		log.Println("db: error saving backtest result:", err)
	}
	return res, err
}

func Newave(x, bcur, qcur string,
	tf, tf2 backtest.Timeframe,
	macdFast, macdSlow strategy.MACDOpts,
	tp, sl float64,
	from, to time.Time) *NewaveConfig {
	return &NewaveConfig{
		Source: backtest.Source{
			Exchange: x,
			Base:     bcur, Quote: qcur, Timeframe: tf, From: from, To: to,
		},
		Profile: trading.Profile{
			TakeProfit: tp,
			StopLoss:   sl,
		},
		TimeframeSlow: tf2,
		MACDFast:      macdFast,
		MACDSlow:      macdSlow,
	}
}

func (n NewaveConfig) Backtest() (*NewaveResult, error) {
	if !n.Timeframe.IsValid() {
		return nil, fmt.Errorf("invalid tf: %s", n.Timeframe)
	}
	if !n.TimeframeSlow.IsValid() {
		return nil, fmt.Errorf("invalid tf2: %s", n.TimeframeSlow)
	}

	histFast, err := backtest.LoadHistorical(db, n.Exchange, n.Base, n.Quote, n.Timeframe, n.From, n.To)
	if err != nil {
		return nil, err
	}
	histSlow, err := backtest.LoadHistorical(db, n.Exchange, n.Base, n.Quote, n.TimeframeSlow, n.From, n.To)
	if err != nil {
		return nil, err
	}

	// set actual n.From after LoadHistorical
	n.From = histFast.From
	n.To = histFast.To

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
	var result = NewaveResult{
		Config: n,
		Result: backtest.Result{
			From: n.From,
			To:   n.To,
		},
	}
	var pos *trading.Position

	j := 0
	var sig1, sig2, last strategy.Signal
	for _, x := range histFast.Data {
		// Positions management ?
		// are we closing ?
		if pos != nil && pos.Active() {
			potentialNet := pos.NetOnClose(x.Close)

			// take profit reached
			if potentialNet > 0 && potentialNet > n.TakeProfit*pos.Cost() {
				pos.PaperCloseAt(x.Close, x.Timestamp.T())
			}
			// stop loss reached
			if potentialNet < 0 && potentialNet < -n.StopLoss*pos.Cost() {
				pos.PaperCloseAt(x.Close, x.Timestamp.T())
			}

			if pos.State == trading.Closed {
				trades.WithLabelValues("sell", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
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
			signals.WithLabelValues("fast", sig1.Action.String()).Inc()
		}
		if sig2 != strategy.NoSignal {
			signals.WithLabelValues("slow", sig1.Action.String()).Inc()
		}

		tt := trigger.Action == strategy.Buy
		if trigger != strategy.NoSignal && trigger.Action != last.Action {
			if chartFlag {
				chart.AddSignal(trigger.Time, tt, true, 0)
			}
			signals.WithLabelValues("newave", trigger.Action.String()).Inc()
			last = trigger

			// buy signal -> open position
			if tt && (pos == nil || pos.State == trading.Closed) {
				pos = trading.NewPosition(x.Timestamp.T(), n.Base, n.Quote, trading.Long)
				pos.PaperBuyAt(k/x.Close, x.Close, x.Timestamp.T())
				trades.WithLabelValues("buy", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
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
func (nwr NewaveResult) Digest() (id string, data []byte, err error) {
	return _db.JSONDigest(
		fmt.Sprintf("%s:%s%s", NewavePrefix, nwr.Config.Base, nwr.Config.Quote),
		nwr,
	)
}

func (n NewaveConfig) String() string {
	return fmt.Sprintf("%8s - %s - %s to %s - macd(%s, %s) & macd(%s, %s) - tp %.1f%% sl %.1f%%",
		fmt.Sprint(n.Base, n.Quote), n.Timeframe,
		n.From.Format("02/01/06"), n.To.Format("02/01/2006"),
		n.Timeframe, n.MACDFast, n.TimeframeSlow, n.MACDSlow,
		n.TakeProfit*100, -n.StopLoss*100,
	)
}

func (nwr NewaveResult) String() string {
	return fmt.Sprintf("%s -> %s", nwr.Config, nwr.Result)
}

//func (n *NewaveConfig) Optimize(maxIterations int) (*NewaveResult, error) {
//	r0, err := n.Backtest()
//	if err != nil {
//		return r0, err
//	}
//
//	var top _db.TopZScorer
//	var best *NewaveResult
//	var bestHash string
//	var stale int
//	for seq := 0; ; seq++ {
//		if seq > maxIterations {
//			return best, nil
//		}
//
//		r1, err := r0.Config.Tune(seq).Backtest()
//		if err != nil {
//			return best, err
//		}
//		r2, err := r0.Config.Tune(seq).Backtest()
//		if err != nil {
//			return best, err
//		}
//
//		top = _db.TopZScorer{r0, r1, r2}
//		sort.Sort(top)
//		if best == nil || top[0].ZScore() > best.ZScore() {
//			best = top[0].(*NewaveResult)
//			bestHash, _, err = best.Digest()
//			if err != nil {
//				log.Printf("best.Digest: %s", err)
//			} else {
//				log.Printf("new best: %s - %s", best, bestHash)
//			}
//			stale = 0
//			r0 = best
//		} else {
//			if stale % 3 == 1 {
//				r0 = r1
//			} else if stale % 3 == 2 {
//				r0 = r2
//			}
//			stale++
//			log.Printf("stale %d", stale)
//		}
//	}
//}

//var (
//	// fib levels
//	fibQuotients = []float64{
//		1: .236,
//		2: .382,
//		3: .5,
//		4: .618,
//	}
//
//	ftoi = func(f float64) int {
//		return int(f)
//	}
//
//	macdConfigs = []strategy.MACDOpts{
//		0: {12, 26, 9},
//		1: {12 - ftoi(12*.236), 26 - ftoi(26*.236), 9 - ftoi(9*.236)},
//		2: {12 + ftoi(12*.236), 26 + ftoi(26*.236), 9 + ftoi(9*.236)},
//		3: {12 - ftoi(12*.382), 26 - ftoi(26*.382), 9 - ftoi(9*.382)},
//		4: {12 + ftoi(12*.382), 26 + ftoi(26*.382), 9 + ftoi(9*.382)},
//		5: {12 - ftoi(12*.618), 26 - ftoi(26*.618), 9 - ftoi(9*.618)},
//		6: {12 + ftoi(12*.618), 26 + ftoi(26*.618), 9 + ftoi(9*.618)},
//	}
//)
//
//func (n *NewaveConfig) Tune(seq int) *NewaveConfig {
//	// update tune state, which indicates how the next
//	// config will be generated.
//
//	// swap sign
//	sign := !n.tuneState.sign
//	n.tuneState.sign = sign
//
//	// inc fib level
//	fib := (n.tuneState.fibIndex + 1) % len(fibQuotients)
//	n.tuneState.fibIndex = fib
//
//	// inc macd config index
//	macd := (n.tuneState.macdIndex + 1) % len(macdConfigs)
//	n.tuneState.macdIndex = macd
//
//	// prepare fib quotient
//	fibQ := fibQuotients[fib]
//	if sign {
//		fibQ = -fibQ
//	}
//
//	// copy current config
//	cfg := *n
//
//	// sequentially change parameters: StopLoss, TakeProfit
//	switch seq % 2 {
//	case 0:
//		delta := fibQ * cfg.StopLoss
//		old := cfg.StopLoss
//		cfg.StopLoss += delta
//		log.Printf("tuning stop-loss with d(%.3f%%): %.3f -> %.3f", delta, old, cfg.StopLoss)
//	//case 4:
//	//	old := cfg.MACDFast
//	//	cfg.MACDFast = macdConfigs[macd]
//	//	log.Printf("changing macdFast(%s) from %s to %s", cfg.Timeframe, old, cfg.MACDFast)
//	case 1:
//		delta := fibQ * cfg.TakeProfit
//		old := cfg.TakeProfit
//		cfg.TakeProfit += delta
//		log.Printf("tuning take-profit with d(%.3f%%): %.3f -> %.3f", delta, old, cfg.TakeProfit)
//	//case 8:
//		// swap macd parameters while breaking sign sequence
//		//cfg.MACDSlow, cfg.MACDFast = cfg.MACDFast, cfg.MACDSlow
//		//log.Printf("swapped macds values: (%s, %s) - (%s, %s)", cfg.Timeframe, cfg.MACDFast,
//		//	cfg.TimeframeSlow, cfg.MACDSlow)
//	}
//	return &cfg
//}
