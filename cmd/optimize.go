package cmd

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"
)

var (
	optimizeCmd = &cobra.Command{
		Use:   "optimize",
		Short: "Optimize a strategy",
		Long: `Usage: optimize <hash>

Loads hash result from a previous backtest, and optimize from
there. It can run for a while depending on optimizer config and strat.Backtest..`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hash := args[0]
			if strings.Index(hash, NewavePrefix) == 0 {
				var nwr NewaveResult
				err := db.LoadJSON(hash, &nwr)
				if err != nil {
					return fmt.Errorf("couldn't load %s: %s", hash, err)
				}

				sa := SimulatedAnnealing{
					Steps: 10000,
					TMax:  25000,
					TMin:  2.5,
				}
				cfg := nwr.Config
				best, err := sa.Optimize(&cfg)
				if best != nil {
					res, _ := best.(*NewaveConfig).Backtest()
					log.Printf("best: %s", best.(*NewaveConfig))
					id, err := db.SaveZScorer(res, "annealing")
					if err != nil {
						log.Println("error saving best result:", err)
					} else {
						log.Printf("saved: %s: %f", id, res.ZScore())
					}
				}
				if err != nil {
					return fmt.Errorf("stopped: %s", err)
				}
				return nil
			}
			return fmt.Errorf("unsupported hash prefix: %s", hash)
		},
	}

	saAccepted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "SA accepted",
	})
	saImproved = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "SA improved",
	})
	saTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "SA total",
	})
	saBestImproved = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "SA new bests",
	})
)

func init() {
	rootCmd.AddCommand(optimizeCmd)
	//optimizeCmd.PersistentFlags().IntVarP(
	//	&optimizeMax, "max", "", 100, "maximum number of optimize iterations")
	rand.Seed(time.Now().Unix())
}

type SimulatedAnnealing struct {
	TMax, TMin float64
	Steps      int
}

func (sa *SimulatedAnnealing) Optimize(state AnnealingState) (best AnnealingState, err error) {
	best = state
	bestE := state.Energy()
	prev := state
	prevE := bestE
	tempCoolingFactor := -math.Log(sa.TMax - sa.TMin)
	temp := sa.TMax
	for i := 0; i < sa.Steps; i++ {
		saTotal.Inc()
		temp = sa.TMax * math.Exp(tempCoolingFactor*float64(i)/float64(sa.Steps))
		prev = state
		state = state.Move()
		E := state.Energy()
		dE := E - prevE
		if dE > 0 && math.Exp(-dE/temp) < randRangeF(0, 1.0) {
			// restore
			state = prev
			E = prevE
		} else {
			// accept
			saAccepted.Inc()
			if dE < 0 {
				// improved
				saImproved.Inc()
			}
			if E < bestE {
				// new best
				log.Printf("new best: %.5f%%", -E)
				saBestImproved.Inc()
				bestE = E
				best = state
			}
		}
	}
	return best, nil
}

type AnnealingState interface {
	Move() AnnealingState
	Energy() float64
}

// Newave implementation
func (cfg *NewaveConfig) Move() AnnealingState {
	//var sign = func() int {
	//	if rand.Int() % 2 == 0 {
	//		return -1
	//	} else {
	//		return 1
	//	}
	//}
	//var signF = func() float64 {return float64(sign())}

	next := *cfg
	next.StopLoss += randRangeF(-0.01, 0.01)
	next.TakeProfit += randRangeF(-0.01, 0.01)
	return &next
}

func (cfg NewaveConfig) Energy() float64 {
	res, err := cfg.Backtest()
	if err != nil {
		log.Println("cfg.Backtest():", err)
		return 0
	}
	// in annealing sim, the lesser the energy the better
	return -res.ZScore()
}

// rand helpers

func randRangeF(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

//randRange := func(min, max int) int {
//	return rand.Intn(max - min) + min
//}
