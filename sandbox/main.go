package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/scraper"
	"github.com/rkjdid/gocx/strategy"
	"gonum.org/v1/plot/vg"
	"log"
	"math"
	"os"
	"time"
)

var (
	_ = talib.Exp
	_ = chart.NextLineStyle
	_ = time.Hour
	_ = stats.ExponentialRegression
	_ = json.Marshal

	bcur = flag.String("base", "BTC", "base cur")
	qcur = flag.String("quote", "USD", "quote cur")
	from = flag.String("from", "03-01-2009", "from date: dd-mm-yyyy")
	x    = flag.String("x", "", "exchange to scrape from")
	to   = flag.String("to", "", "to date: dd-mm-yyyy (defaults to time.Now())")
	tf   = flag.String("tf", scraper.TfDay, "minute/hour/day")
	agg  = flag.Int("agg", 1, "aggregate timeframe (e.g. -tf hour -agg 2 for 2h candles)")

	tfrom, tto time.Time
	tformat    = "02-01-2006" // dd-mm-yyyy
	tformatH   = "02-01-2006 15:04"
)

func init() {
	flag.Parse()
	var err error
	tfrom, err = time.Parse(tformat, *from)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing -from: %s\n", err)
		os.Exit(1)
	}
	if *to == "" {
		tto = time.Now()
	} else {
		tto, err = time.Parse(tformat, *to)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parsing -to: %s\n", err)
			os.Exit(1)
		}
	}
	_, ok := scraper.TfToDuration[*tf]
	if !ok {
		fmt.Fprintf(os.Stderr, "invalid timeframe \"%s\"\n", *tf)
		os.Exit(1)
	}
}

func main() {
	data, err := scraper.FetchHistorical(*x, *bcur, *qcur, *tf, *agg, tfrom, tto)
	if err != nil {
		log.Fatal(err)
	}

	// cleanup input data
	data = data.Trim().Clean()

	if len(data) == 0 {
		fmt.Fprintf(os.Stderr, "no data available")
		os.Exit(1)
	}

	// update actual dates used
	tto = time.Time(data[0].Timestamp)
	tfrom = time.Time(data[len(data)-1].Timestamp)

	// set some capital at t0
	b0 := Bank{Quote: 1000}
	b := b0

	var hi string
	if *x != "" {
		hi += *x + ":"
	}
	fmt.Printf("%s%s%s - tf:%s from %s to %s\n",
		hi, *bcur, *qcur, time.Duration(*agg)*scraper.TfToDuration[*tf],
		tto.Format(tformatH), tfrom.Format(tformatH))

	// alma configs
	almaShort, almaLong, almaSigma, almaOffset := 12, 82, 6., 0.6

	// init chart and draw MAs
	chart.SetTitles(fmt.Sprintf("%s:%s%s", *x, *bcur, *qcur), "", "")
	chart.AddOHLCVs(data)
	chart.AddLine(data.ToXYer(talib.Alma(data.Close(), almaShort, almaSigma, almaOffset)),
		fmt.Sprintf("alma%d", almaShort))
	chart.AddLine(data.ToXYer(talib.Alma(data.Close(), almaLong, almaSigma, almaOffset)),
		fmt.Sprintf("alma%d", almaLong))
	h, _ := stats.Mean(stats.Float64Data(data.Close()))
	chart.AddHorizontal(h, "mean")

	// init & feed strategy
	strat := strategy.NewALMACross(almaShort, almaLong)
	for _, v := range data {
		signal := strat.AddTick(v)
		if signal != strategy.NoSignal {
			fmt.Printf("%4s %2.1f @%5.2f - %s\n",
				signal.Action, signal.Strength, v.Close, time.Time(v.Timestamp).Format(tformatH))

			if signal.Action == strategy.Buy {
				b.Base = b.ToBase(v.Close)
				b.Quote = 0
			} else {
				b.Quote = b.ToQuote(v.Close)
				b.Base = 0
			}

			// draw signal
			chart.AddSignal(signal, v.Close)
		}
	}

	t0 := data[0].Close
	tn := data[len(data)-1].Open
	w := (b.ToQuote(tn) / b0.ToQuote(t0) * 100) - 100
	fmt.Printf("Bank  : %.3f%s (%.1f%s)\n", b.ToBase(tn), *bcur, b.ToQuote(tn), *qcur)
	fmt.Printf("Work  : %.2f%% (%.2f%%/day)\n", w, w/(tfrom.Sub(tto).Hours()/24))
	fmt.Printf("B&Hold: %.2f%%\n", tn/t0*100-100)

	// draw chart
	cname := fmt.Sprintf("c%s%s.png", *bcur, *qcur)
	width := vg.Length(math.Max(float64(len(data)), 1200))
	height := width / 1.77
	err = chart.Save(width, height, true, cname)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("saved \"%s\"", cname)
}
