package cmd

import (
	"fmt"
	_db "github.com/rkjdid/gocx/db"
	"github.com/rkjdid/gocx/scraper/binance"
	"github.com/spf13/cobra"
	"log"
)

var (
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
			driver, ok := db.(*_db.RedisDriver)
			if !ok {
				log.Fatalln("strat command only available with _db.RedisDriver")
				return
			}
			conn := driver.Conn
			resp := conn.Cmd("ZREVRANGE", "results", 0, n-1)
			if resp.Err != nil {
				log.Println("redis:", resp.Err)
				return
			}
			strats, err := resp.Array()
			if err != nil {
				log.Println("redis cast array:", err)
				return
			}
			for i, v := range strats {
				var result Newave2Result
				id, err := v.Str()
				if err != nil {
					log.Println("v.Str():", err)
					continue
				}
				err = db.LoadJSON(id, &result)
				if err != nil {
					log.Println("LoadJSON:", err)
					continue
				}
				fmt.Printf("%3d/  %s\n", i+1, result)
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
	marketsCmd.LocalFlags().StringVar(&x, "x", "binance", "exchange to fetch markets from")

	topCmd.AddCommand(strategiesCmd, marketsCmd)
}
