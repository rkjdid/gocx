package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var (
	showCmd = TraverseRunHooks(&cobra.Command{
		Use:   "show",
		Short: "Show item from db",
		Long:  `Show item from db`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			var id = args[0]
			if strings.Index(id, NewavePrefix) == 0 {
				var nwr NewaveResult
				err = db.LoadJSON(id, &nwr)
				if err != nil {
					log.Fatalf("db.LoadJSON: %s", err)
				}
				fmt.Println(nwr.Details())
			} else {
				// generic backtest.Result
				var r backtest.Result
				err = db.LoadJSON(id, &r)
				if err != nil {
					log.Fatalf("db.LoadJSON: %s", err)
				}
				fmt.Println(r)
			}
		},
	})
)
