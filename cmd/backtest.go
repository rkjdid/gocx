package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	_db "github.com/rkjdid/gocx/db"
	"github.com/rkjdid/gocx/ts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"log"
	"strings"
	"time"
)

var (
	chartFlag  bool
	saveFlag   bool
	x          string
	from, to   string
	tfrom, tto time.Time
	tf, tf2    string
	ttf, ttf2  ts.Timeframe
	tp, sl     float64
	cfgHash    string
	source     backtest.Source

	tformat = "02-01-2006"
)

var backtestCmd = TraverseRunHooks(&cobra.Command{
	Use:   "backtest",
	Short: "Backtest a strategy on an asset",
	Long: `When no sub-command is specified,
backtest is used to run again existing results from db.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		ttf, err = ts.ParseTf(tf)
		if err != nil {
			return fmt.Errorf("parsing -tf: %s\n", err)
		}
		if tf2 == "" {
			ttf2 = ts.Timeframe{
				Unit: ttf.Unit,
				N:    ttf.N * 4,
			}
		} else {
			ttf2, err = ts.ParseTf(tf2)
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
		source = backtest.Source{
			Exchange:  x,
			Base:      "",
			Quote:     "",
			From:      tfrom,
			To:        tto,
			Timeframe: ttf,
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// no args defaults to "all" special id
		var id string
		if len(args) == 0 {
			id = "all"
		} else {
			id = args[0]
		}

		// what we do with the result
		printSave := func(res _db.ZScorer, details bool) {
			msg := fmt.Sprint(res)
			if details {
				if v, ok := res.(Details); ok {
					msg = v.Details()
				}
			}
			fmt.Println(msg)

			if saveFlag {
				_, err := db.SaveZScorer(res, zkey)
				if err != nil {
					log.Printf("save: %s", err)
				}
			}
		}

		if id == "all" || id == "top" {
			keys, err := db.ZREVRANGE(zkey, 0, n)
			if err != nil {
				log.Fatalln("db.ZRANGE:", err)
			}
			for _, key := range keys {
				res, err := rerun(key)
				if err != nil {
					log.Println(key, err)
					continue
				}
				printSave(res, false)
			}
		} else {
			res, err := rerun(id)
			if err != nil {
				log.Fatalln(err)
			}
			printSave(res, true)
		}
	},
})

func addSaveFlag(set *pflag.FlagSet) {
	set.BoolVar(&saveFlag, "save", false, "save results to redis")
}

func init() {
	backtestCmd.PersistentFlags().BoolVar(&chartFlag, "chart", false, "chart executions")
	addSaveFlag(backtestCmd.PersistentFlags())
	backtestCmd.PersistentFlags().IntVarP(&n, "n", "n", 10, "backtest top n markets")
	backtestCmd.PersistentFlags().StringVar(&from, "from", "", "from date: dd-mm-yyyy")
	backtestCmd.PersistentFlags().StringVarP(&x, "exchange", "x", "binance", "exchange to scrape from")
	backtestCmd.PersistentFlags().StringVar(&to, "to", "", "to date: dd-mm-yyyy (defaults to time.Now())")
	backtestCmd.PersistentFlags().StringVar(&tf, "tf", ts.TfDay, tfFlagHelper())
	backtestCmd.PersistentFlags().Float64Var(&tp, "tp", 0.1, "take profit")
	backtestCmd.PersistentFlags().Float64Var(&sl, "sl", 0.025, "stop loss")
	backtestCmd.PersistentFlags().StringVar(&cfgHash, "cfg", "",
		"load config values from provided <hash> and use if as default, explicit flags will overwrite default from cfg")

	backtestCmd.AddCommand(newaveCmd)
}

type Details interface {
	Details() string
}

func rerun(key string) (_db.ZScorer, error) {
	var prefix string
	fields := strings.FieldsFunc(key, func(r rune) bool {
		return r == ':'
	})
	if len(fields) > 0 {
		prefix = fields[0]
	}
	switch prefix {
	case NewavePrefix:
		return rerunNewave(key)
	}
	return nil, fmt.Errorf("unsupported hash prefix: %s", prefix)
}

func rerunNewave(key string) (*NewaveResult, error) {
	var nwr NewaveResult
	err := db.LoadJSON(key, &nwr)
	if err != nil {
		return nil, fmt.Errorf("LoadJSON: %s", err)
	}
	cfg := nwr.Config
	if cfgHash != "" {
		var res NewaveResult
		err := db.LoadJSON(cfgHash, &res)
		if err != nil {
			log.Fatalf("error loading config: %s", err)
		}
		cfg = res.Config
	}
	return rerunNewaveWith(nwr, cfg)
}

func rerunNewaveWith(nwr NewaveResult, cfg NewaveConfig) (*NewaveResult, error) {
	cfg.Source = nwr.Config.Source
	return cfg.Backtest()
}

func tfFlagHelper() string {
	return fmt.Sprintf("<unit>[<n>] with <n> positive int (default 1) and <unit> in %v", ts.TfToDuration)
}
