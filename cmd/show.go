package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/backtest"
	"github.com/spf13/cobra"
	"strings"
)

var (
	showCmd = TraverseRunHooks(&cobra.Command{
		Use:   "show",
		Short: "Show item from db",
		Long:  `Show item from db`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var result interface{}
			var id = args[0]
			if strings.Index(id, NewavePrefix) == 0 {
				var nwr NewaveResult
				err = db.LoadJSON(id, &nwr)
				if err != nil {
					return fmt.Errorf("db.LoadJSON: %s", err)
				}
				for _, v := range nwr.Positions {
					fmt.Println(v)
				}
				fmt.Println(nwr)
				return nil
			} else {
				// generic backtest.Result
				var r backtest.Result
				err = db.LoadJSON(id, &r)
				if err != nil {
					return fmt.Errorf("db.LoadJSON: %s", err)
				}
				result = r
			}
			fmt.Println(result)
			return nil
		},
	})
)
