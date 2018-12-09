package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkjdid/gocx/trading/strategy"
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
		Run: func(cmd *cobra.Command, args []string) {
			hash := args[0]
			if strings.Index(hash, NewavePrefix) == 0 {
				var nwr NewaveResult
				err := db.LoadJSON(hash, &nwr)
				if err != nil {
					log.Fatalf("couldn't load %s: %s", hash, err)
				}
				rank0, _ := db.ZRANK(zkey, hash)
				log.Printf("initial state: %s", nwr.String())
				log.Printf("rank %d", rank0)

				sa := SimulatedAnnealing{
					Steps: 10000,
					TMax:  25000,
					TMin:  2.5,
				}
				cfg := nwr.Config
				res, err := sa.Optimize(&NewaveResult{Config: cfg})
				if res != nil {
					best := res.(*NewaveResult)
					hash, _, err := best.Digest()
					if err != nil {
						log.Fatalf("couldn't digest result: %s", err)
					}
					_, err = db.SaveZScorer(best, "optimized")
					if err != nil {
						log.Println("redis set:", err)
					}
					log.Printf("best: %s", best)
					rank, _ := db.ZRANK(zkey, hash)
					log.Printf("rank %d", rank)
					rawJson, _ := json.MarshalIndent(best, "    ", "  ")
					log.Printf("\n%s", rawJson)
					fmt.Println(best.Details())
				}
				if err != nil {
					log.Fatalf("stopped: %s", err)
				}
				return
			}
			log.Fatalf("unsupported hash prefix: %s", hash)
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
				log.Printf("new best: %.5f%%", -E*100)
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

// Newave implementation for AnnealingState

func (nwr *NewaveResult) Move() AnnealingState {
	next := *nwr

	// todo explore timeframes space also ?

	// risk parameters search space
	next.Config.StopLoss += randRangeF(-0.01, 0.01)
	next.Config.TakeProfit += randRangeF(-0.01, 0.01)
	fixRangeF(&next.Config.StopLoss, 0.01, .618)
	fixRangeF(&next.Config.TakeProfit, 0.05, .618)

	// macds parameters search space
	for _, opts := range []*strategy.MACDOpts{&next.Config.MACDFast, &next.Config.MACDSlow} {
		opts.Fast += randRange(-1, 1)
		opts.Slow += randRange(-1, 1)
		opts.SignalPeriod += randRange(-1, 1)
		fixRange(&opts.Fast, 2, 21)
		fixRange(&opts.Slow, 13, 89)
		fixRange(&opts.SignalPeriod, 2, 34)

		// swap values that crossed for macd.slow/fast
		if opts.Fast > opts.Slow {
			opts.Fast, opts.Slow = opts.Slow, opts.Fast
		}
	}

	return &next
}

func (nwr *NewaveResult) Energy() float64 {
	res, err := nwr.Config.Backtest()
	if err != nil {
		log.Println("nwr.Backtest():", err)
		return 0
	}
	*nwr = *res
	// in annealing sim, the lesser the energy the better
	return -nwr.ZScore()
}

// helpers

func randRangeF(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randRange(min, max int) int {
	return rand.Intn(max-min) + min
}

// fixRangeF updates f if it is out-of-bond
func fixRangeF(v *float64, min, max float64) {
	if *v < min {
		//*v = movOppositeF(*v, min)
		//*v = midLinearF(min, max)
		*v = midLog2F(min, max)
	} else if *v > max {
		//*v = movOppositeF(*v, max)
		//*v = midLinearF(min, max)
		*v = midLog2F(min, max)
	}
}

// fixRange is like fixRangeF but for ints
func fixRange(v *int, min, max int) {
	if *v < min {
		//*v = movOpposite(*v, min)
		//*v = midLinear(min, max)
		*v = midLog2(min, max)
	} else if *v > max {
		//*v = movOpposite(*v, max)
		//*v = midLinear(min, max)
		*v = midLog2(min, max)
	}
}

func movOpposite(v int, bond int) int {
	return 2*bond - v
}

func movOppositeF(v float64, bond float64) float64 {
	return 2*bond - v
}

func midLinearF(min, max float64) float64 {
	return (min + max) / 2
}

func midLog2F(min, max float64) float64 {
	logMin := math.Log2(min)
	logMax := math.Log2(max)
	return math.Pow(2, logMin+(logMax-logMin)/2)
}

func midLinear(min, max int) int {
	return (min + max) / 2
}

func midLog2(min, max int) int {
	logMin := math.Log2(float64(min))
	logMax := math.Log2(float64(max))
	return int(math.Pow(2, logMin+(logMax-logMin)/2))
}
