package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/rkjdid/gocx/backtest/scraper/binance"
	"github.com/rkjdid/gocx/chart"
	_db "github.com/rkjdid/gocx/db"
	"github.com/rkjdid/gocx/trading"
	"github.com/rkjdid/gocx/trading/strategy"
	"github.com/rkjdid/gocx/ts"
	"github.com/rkjdid/gocx/util"
	"github.com/spf13/cobra"
	"log"
	"math"
)

const NewavePrefix = "newave"

type NewaveConfig struct {
	backtest.Source
	trading.Profile
	strategy.NewaveOpts
}

var (
	defaultMACD = strategy.MACDOpts{12, 26, 9, ts.Timeframe{}}

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
			macdSlow := defaultMACD
			macdSlow.Timeframe = ttf
			macdFast := defaultMACD
			macdFast.Timeframe = ttf2
			newaveBaseCfg := Newave(source, macdSlow, macdFast, tp, sl)
			if cfgHash != "" {
				var res NewaveResult
				err := db.LoadJSON(cfgHash, &res)
				if err != nil {
					log.Fatalf("error loading config: %s", err)
				}
				newaveBaseCfg = res.Config

				// overwrite conf values with flag value, if explicitly set
				if cmd.Flags().Changed("tf") {
					newaveBaseCfg.Fast.Timeframe = ttf
				}
				if cmd.Flags().Changed("tf2") {
					newaveBaseCfg.Slow.Timeframe = ttf2
				}
				if cmd.Flags().Changed("x") {
					newaveBaseCfg.Exchange = x
				}
				if cmd.Flags().Changed("from") {
					newaveBaseCfg.From = tfrom
				}
				if cmd.Flags().Changed("to") {
					newaveBaseCfg.To = tto
				}
				if cmd.Flags().Changed("tp") {
					newaveBaseCfg.TakeProfit = tp
				}
				if cmd.Flags().Changed("sl") {
					newaveBaseCfg.StopLoss = sl
				}
			}

			if len(args) == 0 {
				NewaveTop(newaveBaseCfg, n)
			} else if len(args) == 2 {
				_, _ = RunNewaveFor(newaveBaseCfg, args[0], args[1])
			}
		},
	})
)

func init() {
	newaveCmd.Flags().StringVar(&tf2, "tf2", "", tfFlagHelper())
}

func NewaveTop(cfg NewaveConfig, n int) {
	tickers, err := binance.FetchTopTickers("", "BTC")
	if err != nil {
		log.Fatal(err)
	}
	if n <= 0 {
		n = len(tickers)
	}
	for _, v := range tickers[:n] {
		_, _ = RunNewaveFor(cfg, v.Base, v.Quote)
	}
}

func RunNewave(cfg NewaveConfig) (*NewaveResult, error) {
	res, err := cfg.Backtest()
	if err != nil {
		log.Printf("Newave %s:%s%s - %s", x, cfg.Base, cfg.Quote, err)
		return nil, err
	}
	fmt.Println(res.Details())

	if saveFlag {
		_, err = db.SaveZScorer(res, zkey)
		if err != nil {
			log.Println("db: error saving backtest result:", err)
		}
	}
	return res, err
}

func RunNewaveFor(cfg NewaveConfig, bcur, qcur string) (*NewaveResult, error) {
	cfg.Base = bcur
	cfg.Quote = qcur
	return RunNewave(cfg)
}

func Newave(source backtest.Source,
	macdFast, macdSlow strategy.MACDOpts,
	tp, sl float64) NewaveConfig {
	return NewaveConfig{
		Source: source,
		Profile: trading.Profile{
			TakeProfit: tp,
			StopLoss:   sl,
		},
		NewaveOpts: strategy.NewaveOpts{
			Slow: macdSlow,
			Fast: macdFast,
		},
	}
}

func (n NewaveConfig) Backtest() (*NewaveResult, error) {
	if !n.Fast.Timeframe.IsValid() {
		return nil, fmt.Errorf("invalid tf: %s", n.Fast.Timeframe)
	}
	if !n.Slow.Timeframe.IsValid() {
		return nil, fmt.Errorf("invalid tf2: %s", n.Slow.Timeframe)
	}

	histFast, err := backtest.LoadHistorical(db, n.Exchange, n.Base, n.Quote, n.Fast.Timeframe, n.From, n.To)
	if err != nil {
		return nil, err
	}
	histSlow, err := backtest.LoadHistorical(db, n.Exchange, n.Base, n.Quote, n.Slow.Timeframe, n.From, n.To)
	if err != nil {
		return nil, err
	}

	// init DataSource
	source := backtest.NewHistoricalPair(histFast, histSlow)
	// set actual bondaries after LoadHistorical
	n.From, n.To = source.Bondaries()

	// init chart
	if chartFlag {
		chart.SetTitles(fmt.Sprintf("%s:%s%s", n.Exchange, n.Base, n.Quote), "", "")
		chart.AddOHLCVs(histSlow.Data)
	}

	// init strategy
	newaveStrat := n.NewNewave()

	// init capital & resulty
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
	var last strategy.Signal

	for x := range source.Feed() {
		// set price & time on paper
		if pos != nil {
			pos.SetTick(x.OHLCV)
		}
		// manage position
		if pos != nil && pos.Active() {
			potentialNet := pos.NetOnClose()

			// take profit reached
			if potentialNet > 0 && potentialNet > n.TakeProfit*pos.Cost() {
				_ = pos.Close()
			}
			// stop loss reached
			if potentialNet < 0 && potentialNet < -n.StopLoss*pos.Cost() {
				_ = pos.Close()
			}

			if pos.State == trading.Closed {
				k += pos.Net()
				if chartFlag {
					chart.AddSignal(x.Timestamp.T(), false, true, x.Close)
				}
			}
		}

		// feed strat
		newaveStrat.AddTick(x)

		// signal changed
		if s := newaveStrat.Signal(); s.Action != last.Action {
			last = s

			if last.Action != strategy.None {
				if pos != nil && pos.State != trading.Closed {
					continue
				}

				// buy signal -> open long
				if last.Action == strategy.Buy {
					pos = trading.NewPosition(broker, n.Base, n.Quote, trading.Long)
					pos.SetTick(x.OHLCV)
					_ = pos.MarketBuy(k / x.Close)
					result.Positions = append(result.Positions, pos)

					if chartFlag {
						chart.AddSignal(x.Timestamp.T(), true, true, x.Close)
					}
				}
			}
		}
	}

	result.UpdateScore()

	// draw chart
	if chartFlag {
		_ = newaveStrat.Fast.Draw()
		chart.NextLineTheme()
		_ = newaveStrat.Slow.Draw()
		cname := fmt.Sprintf("img/newave_%s%s.png", n.Base, n.Quote)
		width := math.Max(float64(len(histFast.Data)), 1200)
		height := width / 1.77

		err = chart.Save(width, height, false, cname)
		if err != nil {
			return &result, err
		}
		log.Printf("saved \"%s\"", cname)
	}

	return &result, nil
}

func (n NewaveConfig) String() string {
	return fmt.Sprintf("macd(%s, %s) & macd(%s, %s) - tp %.1f%% sl %.1f%%",
		n.Fast.Timeframe, n.Fast, n.Slow.Timeframe, n.Slow,
		n.TakeProfit*100, -n.StopLoss*100,
	)
}

type NewaveResult struct {
	Config NewaveConfig
	backtest.Result
	Id string
}

// Digest is db.Digester implementation with json data and a id-hash.
func (nwr *NewaveResult) Digest() (id string, data []byte, err error) {
	id, data, err = _db.JSONDigest(
		fmt.Sprintf("%s:%s%s", NewavePrefix, nwr.Config.Base, nwr.Config.Quote),
		nwr,
	)
	nwr.Id = id
	return
}

func (nwr NewaveResult) String() string {
	if nwr.Id == "" {
		_, _, _ = nwr.Digest()
	}
	return fmt.Sprintf("%s | %s | %s | %s", nwr.Id, nwr.Config.Source, nwr.Config, nwr.Result)
}

func (nwr NewaveResult) Details() string {
	s := ""
	for _, p := range nwr.Positions {
		s += fmt.Sprintln(p)
	}
	hash, _, _ := nwr.Digest()
	return s + fmt.Sprintln(nwr) + fmt.Sprintln("id:", hash)
}

// SA implementation

func (nwr *NewaveResult) Move() AnnealingState {
	next := *nwr

	// todo explore timeframes space also ?

	// risk parameters search space
	next.Config.StopLoss += util.RandRangeF(-0.01, 0.01)
	next.Config.TakeProfit += util.RandRangeF(-0.01, 0.01)
	util.FixRangeLinearF(&next.Config.StopLoss, 0.01, .618)
	util.FixRangeLinearF(&next.Config.TakeProfit, 0.05, .618)

	// macds parameters search space
	for _, opts := range []*strategy.MACDOpts{&next.Config.Fast, &next.Config.Slow} {
		opts.Fast += util.RandRange(-1, 1)
		opts.Slow += util.RandRange(-1, 1)
		opts.SignalPeriod += util.RandRange(-1, 1)
		util.FixRangeLinear(&opts.Fast, 2, 21)
		util.FixRangeLinear(&opts.Slow, 13, 89)
		util.FixRangeLinear(&opts.SignalPeriod, 2, 34)

		// swap values that crossed for macd.slow/fast
		if opts.Fast > opts.Slow {
			opts.Fast, opts.Slow = opts.Slow, opts.Fast
		}
	}

	return &next
}

func (nwr *NewaveResult) Energy() float64 {
	res, err := nwr.Config.Backtest()
	if err != nil {
		log.Println("nwr.Backtest():", err)
		return 0
	}
	*nwr = *res
	// in annealing sim, the lesser the energy the better
	return -nwr.ZScore()
}
