package cmd

import (
	"fmt"
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
)

func init() {
	addSaveFlag(snapshotCmd.PersistentFlags())
}
