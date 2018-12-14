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
	Id string
}

var (
	defaultMACD = strategy.MACDOpts{12, 26, 9}

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
			newaveBaseCfg := Newave(source, ttf2, defaultMACD, defaultMACD, tp, sl)
			if cfgHash != "" {
				var res NewaveResult
				err := db.LoadJSON(cfgHash, &res)
				if err != nil {
					log.Fatalf("error loading config: %s", err)
				}
				newaveBaseCfg = res.Config

				// overwrite conf values with flag value, if explicitly set
				if cmd.Flags().Changed("tf") {
					newaveBaseCfg.Timeframe = ttf
				}
				if cmd.Flags().Changed("tf2") {
					newaveBaseCfg.TimeframeSlow = ttf2
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
				RunNewaveFor(newaveBaseCfg, args[0], args[1])
			}
		},
	})
)

func init() {
	newaveCmd.Flags().StringVar(&tf2, "tf2", "", tfFlagHelper())
	// todo add macdFast, macdSlow flags
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
		RunNewaveFor(cfg, v.Base, v.Quote)
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

func Newave(source backtest.Source, tf2 backtest.Timeframe,
	macdFast, macdSlow strategy.MACDOpts,
	tp, sl float64) NewaveConfig {
	return NewaveConfig{
		Source: source,
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
	macdFast := n.MACDFast.NewMACDCross()
	macdSlow := n.MACDSlow.NewMACDCross()
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

		macdFast.AddTick(x)
		sig1 = macdFast.Signal()
		if len(histSlow.Data) > j+1 && x.Timestamp.T().After(histSlow.Data[j+1].Timestamp.T()) {
			j += 1
			macdSlow.AddTick(histSlow.Data[j])
			sig2 = macdSlow.Signal()
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
func (nwr *NewaveResult) Digest() (id string, data []byte, err error) {
	id, data, err = _db.JSONDigest(
		fmt.Sprintf("%s:%s%s", NewavePrefix, nwr.Config.Base, nwr.Config.Quote),
		nwr,
	)
	nwr.Id = id
	return
}

func (n NewaveConfig) String() string {
	return fmt.Sprintf("macd(%s, %s) & macd(%s, %s) - tp %.1f%% sl %.1f%%",
		n.Timeframe, n.MACDFast, n.TimeframeSlow, n.MACDSlow,
		n.TakeProfit*100, -n.StopLoss*100,
	)
}

func (nwr NewaveResult) String() string {
	if nwr.Id == "" {
		nwr.Digest()
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
