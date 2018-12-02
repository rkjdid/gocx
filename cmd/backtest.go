package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/scraper"
	"github.com/spf13/cobra"
	"time"
)

var (
	chartFlag  bool
	x          string
	from, to   string
	tfrom, tto time.Time
	tf         string
	agg        int

	tformat = "02-01-2006"
)

var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "Backtest a strategy on an asset",
	Long:  `Needs a strategy, a config, an asset, a timeframe.. stuff like that`,
	Args: func(cmd *cobra.Command, args []string) error {
		var err error
		tfrom, err = time.Parse(tformat, from)
		if err != nil {
			return fmt.Errorf("parsing -from: %s\n", err)
		}
		if to == "" {
			tto = time.Now()
		} else {
			tto, err = time.Parse(tformat, to)
			if err != nil {
				return fmt.Errorf("parsing -to: %s\n", err)
			}
		}
		_, ok := scraper.TfToDuration[tf]
		if !ok {
			return fmt.Errorf("invalid timeframe \"%s\"\n", tf)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backtestCmd)
	backtestCmd.PersistentFlags().BoolVar(&chartFlag, "chart", false, "chart executions")
	backtestCmd.PersistentFlags().IntVarP(&n, "n", "n", 10, "backtest top n markets")
	backtestCmd.PersistentFlags().StringVar(&from, "from", "03-01-2009", "from date: dd-mm-yyyy")
	backtestCmd.PersistentFlags().StringVarP(&x, "exchange", "x", "binance", "exchange to scrape from")
	backtestCmd.PersistentFlags().StringVar(&to, "to", "", "to date: dd-mm-yyyy (defaults to time.Now())")
	backtestCmd.PersistentFlags().StringVar(&tf, "tf", scraper.TfDay, fmt.Sprintf("one of: %v", scraper.TfToDuration))
	backtestCmd.PersistentFlags().IntVar(&agg, "agg", 1, "aggregate tf (e.g. -tf hour -agg 2 for 2h candles)")
}
