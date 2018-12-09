package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/rkjdid/gocx/scraper/binance"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var (
	script bool

	topCmd = TraverseRunHooks(&cobra.Command{
		Use:   "top",
		Short: "Display top items",
		Long:  `Display best scoring strategies in redisConn or binance top markets`,
	})

	strategiesCmd = TraverseRunHooks(&cobra.Command{
		Use:   "strat",
		Short: "Display best scoring backtest executions",
		Long:  `ZREVRANGE on the sorted set holding strat backtest and display corresponding results`,
		Run: func(cmd *cobra.Command, args []string) {
			keys, err := db.ZREVRANGE(zkey, 0, n-1)
			if err != nil {
				log.Fatalf("db.ZREVRANGE: %s", err)
			}
			for i, key := range keys {
				if script {
					fmt.Println(key)
					continue
				}

				var result interface{}
				if strings.Index(key, NewavePrefix) == 0 {
					var nwr NewaveResult
					err = db.LoadJSON(key, &nwr)
					if err != nil {
						log.Println("LoadJSON:", err)
						continue
					}
					result = nwr
				} else {
					// generic backtest.Result
					var r backtest.Result
					err = db.LoadJSON(key, &r)
					if err != nil {
						log.Println("LoadJSON:", err)
						continue
					}
					result = r
				}
				fmt.Printf("%3d/ %s %s\n", i+1, key, result)
			}
		},
	})

	marketsCmd = TraverseRunHooks(&cobra.Command{
		Use:   "markets",
		Short: "Display top volume markets",
		Long:  `Fetch markets from binance API and order them by volume desc`,
		Run: func(cmd *cobra.Command, args []string) {
			tickers, err := binance.FetchTopTickers("", "BTC")
			if err != nil {
				log.Fatal(err)
			}
			if n <= 0 {
				n = len(tickers)
			}
			for i, t := range tickers[:n] {
				fmt.Printf("%3d/  %8s vol: %9.2f\n", i+1, t.Symbol, t.QuoteVolume)
			}
		},
	})
)

func init() {
	topCmd.PersistentFlags().IntVarP(&n, "n", "n", 10, "top n markets/executions")
	topCmd.PersistentFlags().BoolVarP(&script, "script", "s", false, "print in a script friendly manner, if possible")
	marketsCmd.LocalFlags().StringVarP(&x, "exchange", "x", "binance", "exchange to fetch markets from")

	topCmd.AddCommand(strategiesCmd, marketsCmd)
}
