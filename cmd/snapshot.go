package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/trading"
	"github.com/spf13/cobra"
	"log"
)

var (
	snapshotCmd = TraverseRunHooks(&cobra.Command{
		Use:   "snapshot",
		Short: "Get the balances snapshot of crypto account",
		Long:  `Displays performance at time of request for given account, optionally save`,
		Run: func(cmd *cobra.Command, args []string) {
			s, err := broker.Snapshot()
			if err != nil {
				log.Fatalf("broker.Snapshot(): %s", err)
			}
			fmt.Println(s)
			if saveFlag {
				_, err := db.SaveZScorer(s, s.Account)
				if err != nil {
					log.Printf("error saving snapshot: %s", err)
				}
			}
		},
	})

	chartCmd = TraverseRunHooks(&cobra.Command{
		Use:   "chart",
		Short: "Chart balance(s)",
		Long:  `Default to "all" balances`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ids, err := db.ZREVRANGE(args[0], 0, -1)
			if err != nil {
				log.Fatalf("db error: %s", err)
			}
			var sn []trading.Snapshot
			assetMap := make(map[string]bool)
			for _, id := range ids {
				var s trading.Snapshot
				err := db.LoadJSON(id, &s)
				if err != nil {
					log.Printf("db load (%s): %s", id, err)
					continue
				}
				sn = append(sn, s)
				for i := range s.Balances {
					assetMap[i] = true
				}
			}

			var assets []string
			if len(args) == 1 {
				for key := range assetMap {
					assets = append(assets, key)
				}
			} else {
				assets = args[1:]
			}
			for _, asset := range assets {
				ah := trading.AssetHistoryBTC(trading.ExtractAssetHistory(sn, asset))
				chart.SetRanges(ah.Range())
				chart.AddLine(ah, asset)
			}

			chart.SetTitles(fmt.Sprintf("%s balances", args[0]), "", "")

			w := 400 + float64(len(sn))*15
			h := w / 1.77
			fname := fmt.Sprintf("balances_%s.png", args[0])
			err = chart.Save(w, h, true, fname)
			if err != nil {
				log.Fatal("error saving chart: ", err)
			}
			log.Printf("saved %s", fname)
		},
	})
)

func init() {
	addSaveFlag(snapshotCmd.PersistentFlags())
}
