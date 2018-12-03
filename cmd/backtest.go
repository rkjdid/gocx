package cmd

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkjdid/gocx/backtest"
	"github.com/rkjdid/gocx/scraper"
	"github.com/spf13/cobra"
	"time"
)

var (
	chartFlag  bool
	x          string
	from, to   string
	tfrom, tto time.Time
	tf, tf2    string
	ttf, ttf2  backtest.Timeframe

	tformat = "02-01-2006"

	// prometheus metrics
	sigCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signal", Help: "signals",
	}, []string{"name", "action"})

	tradeCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trade", Help: "trades",
	}, []string{"direction", "quantity", "price"})
)

var backtestCmd = TraverseRunHooks(&cobra.Command{
	Use:   "backtest",
	Short: "Backtest a strategy on an asset",
	Long:  `Needs a strategy, a config, an asset, a timeframe.. stuff like that`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if from != "" {
			tfrom, err = time.Parse(tformat, from)
			if err != nil {
				return fmt.Errorf("parsing -from: %s\n", err)
			}
		}
		if to == "" {
			tto = time.Now()
		} else {
			tto, err = time.Parse(tformat, to)
			if err != nil {
				return fmt.Errorf("parsing -to: %s\n", err)
			}
		}
		ttf, err = backtest.ParseTf(tf)
		if err != nil {
			return fmt.Errorf("parsing -tf: %s\n", err)
		}
		if tf2 == "" {
			ttf2 = backtest.Timeframe{
				Unit: ttf.Unit,
				N:    ttf.N * 4,
			}
		} else {
			ttf2, err = backtest.ParseTf(tf2)
			if err != nil {
				return fmt.Errorf("parsing -tf2: %s\n", err)
			}
		}

		return nil
	},
})

func init() {
	backtestCmd.PersistentFlags().BoolVar(&chartFlag, "chart", false, "chart executions")
	backtestCmd.PersistentFlags().IntVarP(&n, "n", "n", 10, "backtest top n markets")
	backtestCmd.PersistentFlags().StringVar(&from, "from", "", "from date: dd-mm-yyyy")
	backtestCmd.PersistentFlags().StringVarP(&x, "exchange", "x", "binance", "exchange to scrape from")
	backtestCmd.PersistentFlags().StringVar(&to, "to", "", "to date: dd-mm-yyyy (defaults to time.Now())")
	backtestCmd.PersistentFlags().StringVar(&tf, "tf", scraper.TfDay, tfFlagHelper())

	backtestCmd.AddCommand(newaveCmd, newave2Cmd)
}

func tfFlagHelper() string {
	return fmt.Sprintf("<unit>[<n>] with <n> positive int (default 1) and <unit> in %v", scraper.TfToDuration)
}
