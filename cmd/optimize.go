package cmd

import (
	"fmt"
	"github.com/rkjdid/gocx/util"
	"github.com/spf13/cobra"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"
)

var (
	saConfig = SimulatedAnnealing{
		Steps: 2000,
		TMax:  25000,
		TMin:  2500,
	}

	zkeyOptimized string

	optimizeCmd = TraverseRunHooks(&cobra.Command{
		Use:   "optimize",
		Short: "Optimize a strategy",
		Long: `Usage: optimize <hash>

Loads hash result from a previous backtest, and optimize from
there. It can run for a while depending on optimizer config and strat.Backtest..`,
		Args: cobra.MaximumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			forcePaperBroker()
		},
		Run: func(cmd *cobra.Command, args []string) {
			var hash string
			if len(args) == 0 {
				hash = "all"
			} else {
				hash = args[0]
			}
			if hash == "all" {
				keys, err := db.ZREVRANGE(zkey, 0, n)
				if err != nil {
					log.Fatalf("db.ZREVRANGE: %s", err)
				}
				for _, key := range keys {
					if strings.Index(key, NewavePrefix) == 0 {
						t0 := time.Now()
						log.Printf("optimize %s", key)
						err = OptimizeNewave(key)
						if err != nil {
							log.Printf("optimize %s: %s", key, err)
						} else {
							log.Printf("done %s", time.Since(t0))
						}
					} else {
						log.Fatalf("unsupported hash prefix: %s", key)
					}
				}
			} else if strings.Index(hash, NewavePrefix) == 0 {
				err := OptimizeNewave(hash)
				if err != nil {
					log.Fatalf("optimize %s: %s", hash, err)
				}
			} else {
				log.Fatalf("unsupported hash prefix: %s", hash)
			}
		},
	})
)

func init() {
	optimizeCmd.PersistentFlags().IntVarP(
		&n, "n", "n", -1, "optimize top n results")
	optimizeCmd.PersistentFlags().StringVar(&zkeyOptimized, "zkey2", "optimized", "zkey optimized results")
	addSaveFlag(optimizeCmd.PersistentFlags())
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
	tempCoolingFactor := -math.Log2(sa.TMax - sa.TMin)
	temp := sa.TMax

	attempts, accepts, improves, rejects, noBest, noImprove, resets := 0, 0, 0, 0, 0, 0, 0
	ticker := time.NewTicker(time.Second * 20)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			log.Printf("temp: %.2f, total: %d, accepted: %d, improved: %d, rejected: %d, resets: %d",
				temp, attempts, accepts, improves, rejects, resets)
		}
	}()

	for i := 0; i < sa.Steps; i++ {
		attempts++
		noImprove++
		noBest++
		temp = sa.TMax * math.Exp(tempCoolingFactor*float64(i)/float64(sa.Steps))
		prev = state
		state = state.Move()
		E := state.Energy()
		dE := E - prevE
		if dE > 0 && math.Exp(-dE/temp) < util.RandRangeF(0, 1.0) {
			// restore
			rejects++
			state = prev
			E = prevE
		} else {
			// accept
			accepts++
			if dE < 0 {
				// improved
				improves++
				noImprove = 0
			}
			if E < bestE {
				// new best
				log.Printf("new best: %.5f%%", -E*100)
				bestE = E
				best = state
				noBest = 0
			}
		}
		// if no improvement or no new best for a while:
		//   - reset to best state
		//   - set temperature to +5% (more states to visit)
		if noBest > int(.3*float64(sa.Steps)) || noImprove > int(.10*float64(sa.Steps)) {
			best = state
			i += int(.05 * float64(sa.Steps))
		}
	}
	return best, nil
}

type AnnealingState interface {
	Move() AnnealingState
	Energy() float64
}

// OptimizeNewave is the main optimization func for a given NewaveResult hash
func OptimizeNewave(hash string) error {
	var nwr NewaveResult
	err := db.LoadJSON(hash, &nwr)
	if err != nil {
		return fmt.Errorf("couldn't load %s: %s", hash, err)
	}
	rank0, _ := db.ZRANK(zkey, hash)
	log.Printf("initial state: %s", nwr.String())
	log.Printf("rank %d", rank0)

	sa := saConfig
	cfg := nwr.Config
	res, err := sa.Optimize(&NewaveResult{Config: cfg})
	if res != nil {
		best := res.(*NewaveResult)
		hash, _, err := best.Digest()
		if err != nil {
			return fmt.Errorf("couldn't digest result: %s", err)
		}
		if saveFlag {
			_, err = db.SaveZScorer(best, zkeyOptimized)
			if err != nil {
				log.Println("redis save:", err)
			}
		}
		log.Printf("best: %s", best)
		fmt.Println(best.Details())
		rank, _ := db.ZRANK(zkeyOptimized, hash)
		log.Printf("id: %s, rank %d", hash, rank)
	}
	return err
}
