package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/rkjdid/gocx/scraper"
	"github.com/spf13/cobra"
	"log"
	"strings"
	"time"
)

var (
	chartFlag  bool
	x          string
	from, to   string
	tfrom, tto time.Time
	tf, tf2    string
	ttf, ttf2  backtest.Timeframe
	tp, sl     float64

	tformat = "02-01-2006"
)

var backtestCmd = TraverseRunHooks(&cobra.Command{
	Use:   "backtest",
	Short: "Backtest a strategy on an asset",
	Long:  `Needs a strategy, a config, an asset, a timeframe.. stuff like that`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
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
		if from != "" {
			tfrom, err = time.Parse(tformat, from)
			if err != nil {
				return fmt.Errorf("parsing -from: %s\n", err)
			}
		}
		if to == "" {
			tto = time.Now().Truncate(ttf.ToDuration())
		} else {
			tto, err = time.Parse(tformat, to)
			if err != nil {
				return fmt.Errorf("parsing -to: %s\n", err)
			}
		}
		return nil
	},
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// re-run backtest from redis key
		id := args[0]
		if id == "all" {
			keys, err := db.ZREVRANGE(zkey, 0, -1)
			if err != nil {
				log.Fatalln("db.ZRANGE:", err)
			}
			for _, key := range keys {
				res, err := rerunNewave(key)
				if err != nil {
					log.Println(key, err)
					continue
				}
				fmt.Println(res.Details())
				// update db
				_, err = db.SaveZScorer(res, zkey)
				if err != nil {
					log.Printf("save: %s", err)
				}
			}
		} else if strings.Index(id, NewavePrefix+":") == 0 {
			res, err := rerunNewave(id)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println(res.Details())
			// update db
			_, err = db.SaveZScorer(res, zkey)
			if err != nil {
				log.Printf("save: %s", err)
			}
		} else {
			log.Fatalf("unsupported hash prefix: %s", id)
		}
	},
})

func init() {
	backtestCmd.PersistentFlags().BoolVar(&chartFlag, "chart", false, "chart executions")
	backtestCmd.PersistentFlags().IntVarP(&n, "n", "n", 10, "backtest top n markets")
	backtestCmd.PersistentFlags().StringVar(&from, "from", "", "from date: dd-mm-yyyy")
	backtestCmd.PersistentFlags().StringVarP(&x, "exchange", "x", "binance", "exchange to scrape from")
	backtestCmd.PersistentFlags().StringVar(&to, "to", "", "to date: dd-mm-yyyy (defaults to time.Now())")
	backtestCmd.PersistentFlags().StringVar(&tf, "tf", scraper.TfDay, tfFlagHelper())
	backtestCmd.PersistentFlags().Float64Var(&tp, "tp", 0.1, "take profit")
	backtestCmd.PersistentFlags().Float64Var(&sl, "sl", 0.025, "stop loss")

	backtestCmd.AddCommand(newaveCmd)
}

func rerunNewave(key string) (*NewaveResult, error) {
	var nwr NewaveResult
	err := db.LoadJSON(key, &nwr)
	if err != nil {
		return nil, fmt.Errorf("LoadJSON: %s", err)
	}
	return nwr.Config.Backtest()
}

func tfFlagHelper() string {
	return fmt.Sprintf("<unit>[<n>] with <n> positive int (default 1) and <unit> in %v", scraper.TfToDuration)
}
