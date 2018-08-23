package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/scraper"
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
	agg  = flag.Int("agg", 1, "aggregate timeframe (`-tf hour -agg 2` for 2h candles)")

	tfrom, tto time.Time
	tformat    = "02-01-2006" // dd-mm-yyyy
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
	//tfrom = time.Time{}.AddDate(2016, 0, 0)
	//tto = tfrom.AddDate(1, 0, 0)

	data, err := scraper.FetchHistorical(*x, *bcur, *qcur, *tf, *agg, tfrom, tto)
	if err != nil {
		log.Fatal(err)
	}
	data = data.Trim().Clean()

	chart.SetTitles(fmt.Sprintf("%s:%s%s", *x, *bcur, *qcur), "", "")
	chart.AddOHLCVs(data)
	chart.AddLine(data.ToXYer(talib.Alma(data.Close(), 24, 6, 0.6)), "alma24")
	chart.AddLine(data.ToXYer(talib.Alma(data.Close(), 120, 6, 0.6)), "alma120")

	h, _ := stats.Mean(stats.Float64Data(data.Close()))
	chart.AddHorizontal(h, "mean")

	h, _ = stats.Median(stats.Float64Data(data.Close()))
	chart.AddHorizontal(h, "median")

	name := "chart.png"
	width := vg.Length(math.Max(float64(len(data)), 1200))
	height := width / 1.77
	err = chart.Save(width, height, true, name)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("saved \"%s\"", name)
}
