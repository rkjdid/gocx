package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/markcheno/go-talib"
	"github.com/montanaflynn/stats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rkjdid/gocx/chart"
	"github.com/rkjdid/gocx/position"
	"github.com/rkjdid/gocx/scraper"
	"github.com/rkjdid/gocx/strategy"
	"github.com/rkjdid/gocx/ts"
	"gonum.org/v1/plot/vg"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

var (
	_ = talib.Exp
	_ = chart.NextLineStyle
	_ = time.Hour
	_ = stats.ExponentialRegression
	_ = json.Marshal

	bcur       = flag.String("base", "BTC", "base cur")
	qcur       = flag.String("quote", "USD", "quote cur")
	from       = flag.String("from", "03-01-2009", "from date: dd-mm-yyyy")
	x          = flag.String("x", "", "exchange to scrape from")
	to         = flag.String("to", "", "to date: dd-mm-yyyy (defaults to time.Now())")
	tf         = flag.String("tf", scraper.TfDay, "minute/hour/day")
	tf2        = flag.String("tf2", scraper.TfDay, "minute/hour/day")
	agg        = flag.Int("agg", 1, "aggregate tf (e.g. -tf hour -agg 2 for 2h candles)")
	agg2       = flag.Int("agg2", 1, "aggregate tf2")
	prefix     = flag.String("prefix", "", "prefix")
	promBind   = flag.String("prometheus-bind", ":8080", "prometheus bind")
	promHandle = flag.String("prometheus-handle", "/prometheus", "prometheus handle")
	promServer = flag.Bool("prometheus-server", false, "enable prometheus webserver")

	tfrom, tto time.Time
	tformat    = "02-01-2006" // dd-mm-yyyy
	tformatH   = "02-01-2006 15:04"
)

var (
	sigCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signal", Help: "various signals",
	}, []string{"name", "action"})

	tradeCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trade", Help: "trades",
	}, []string{"direction", "quantity", "price"})
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

	prometheus.MustRegister(sigCount, tradeCount)
	if *promServer {
		http.Handle(*promHandle, promhttp.Handler())
		fmt.Printf("localhost%s%s\n", *promBind, *promHandle)
		go log.Fatal(http.ListenAndServe(*promBind, nil))
	}
}

type Historical struct {
	Data      ts.OHLCVs
	From      time.Time
	To        time.Time
	Timeframe time.Duration

	Exchange, Base, Quote string
}

func (h Historical) String() string {
	var hi string
	if h.Exchange != "" {
		hi += h.Exchange + ":"
	}
	return fmt.Sprintf("%s%s%s - tf:%s %6d elements from %s to %s",
		hi, h.Base, h.Quote, h.Timeframe, h.Data.Len(),
		tfrom.Format(tformatH), tto.Format(tformatH))
}

func LoadHistorical(x, bcur, qcur string, tf string, agg int, from, to time.Time) (*Historical, error) {
	data, err := scraper.FetchHistorical(x, bcur, qcur, tf, agg, from, to)
	if err != nil {
		return nil, err
	}
	// cleanup input data
	data = data.Trim().Clean()

	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	h := Historical{
		Data:      data,
		To:        time.Time(data[0].Timestamp),
		From:      time.Time(data[len(data)-1].Timestamp),
		Timeframe: time.Duration(agg) * scraper.TfToDuration[tf],
		Exchange:  x, Base: bcur, Quote: qcur,
	}
	fmt.Println("loaded:", h)
	return &h, nil
}

func Newave(x, bcur, qcur string, tf string, agg int, tf2 string, agg2 int, from, to time.Time) error {
	hist1, err := LoadHistorical(x, bcur, qcur, tf, agg, from, to)
	if err != nil {
		return err
	}
	hist2, err := LoadHistorical(x, bcur, qcur, tf2, agg2, from, to)
	if err != nil {
		return err
	}

	// init chart
	chart.SetTitles(fmt.Sprintf("%s:%s%s", x, bcur, qcur), "", "")
	chart.AddOHLCVs(hist2.Data)

	// init & feed strategy
	macd1 := strategy.NewMACD(12, 26, 9)
	macd2 := strategy.NewMACD(12, 26, 9)
	if hist1.Timeframe > hist2.Timeframe {
		hist1, hist2 = hist2, hist1
		macd1, macd2 = macd2, macd1
	}

	var k0 = 1000.0
	var k = k0
	var positions []*position.Position
	var pos *position.Position

	j := 0
	var sig1, sig2, last strategy.Signal
	for _, x := range hist1.Data {
		// are we closing ?
		if pos != nil && pos.Active() {
			potentialNet := pos.NetOnClose(x.Close)

			// target +20%
			if potentialNet > 0 && potentialNet > 0.25*pos.Cost() {
				pos.PaperCloseAt(x.Close, x.Timestamp.T())
			}
			// stop   -5%
			if potentialNet < 0 && potentialNet < -0.05*pos.Cost() {
				pos.PaperCloseAt(x.Close, x.Timestamp.T())
			}

			if pos.State == position.Closed {
				tradeCount.WithLabelValues("sell", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
				k += pos.Net()
			}
		}

		sig1 = macd1.AddTick(x)
		if len(hist2.Data) > j+1 && x.Timestamp.T().After(hist2.Data[j+1].Timestamp.T()) {
			j += 1
			sig2 = macd2.AddTick(hist2.Data[j])
		}

		var trigger strategy.Signal
		if sig1 != strategy.NoSignal && macd2.LastSignal.Action == sig1.Action {
			trigger = sig1
		} else if sig2 != strategy.NoSignal && macd1.LastSignal.Action == sig2.Action {
			trigger = sig2
		}

		// print signals individually
		t1 := sig1.Action == strategy.Buy
		if sig1 != strategy.NoSignal {
			_ = t1
			//chart.AddSignal(sig1.Time, t1, false, 10000)
			sigCount.WithLabelValues("fast", sig1.Action.String()).Inc()
		}
		t2 := sig2.Action == strategy.Buy
		if sig2 != strategy.NoSignal {
			_ = t2
			//chart.AddSignal(sig2.Time, t2, true, 20000)
			sigCount.WithLabelValues("slow", sig1.Action.String()).Inc()
		}

		tt := trigger.Action == strategy.Buy
		if trigger != strategy.NoSignal && trigger.Action != last.Action {
			//fmt.Printf("%4s @%5.2f - %s\n", trigger.Action, x.Close, time.Time(x.Timestamp).Format(tformatH))
			chart.AddSignal(trigger.Time, tt, true, 0)
			sigCount.WithLabelValues("newave", trigger.Action.String()).Inc()
			last = trigger

			// buy signal
			if tt && (pos == nil || pos.State == position.Closed) {
				pos = position.NewPosition(x.Timestamp.T(), bcur, qcur, position.Long)
				pos.PaperBuyAt(k/x.Close, x.Close, x.Timestamp.T())
				tradeCount.WithLabelValues("buy", fmt.Sprint(pos.Total), fmt.Sprint(x.Close))
				positions = append(positions, pos)
			}
		}
	}

	for _, p := range positions {
		fmt.Println(p)
	}
	fmt.Println()
	fmt.Printf("initial: %f     net: %.2f    work: %.2f%%\n", k0, k, 100*(k/k0-1))

	// draw chart
	macd1.Draw()
	chart.NextLineTheme()
	macd2.Draw()
	cname := fmt.Sprintf("%sc%s%s.png", *prefix, bcur, qcur)
	width := vg.Length(math.Max(float64(len(hist1.Data)), 1200))
	height := width / 1.77

	err = chart.Save(width, height, false, cname)
	if err != nil {
		return err
	}
	log.Printf("saved \"%s\"", cname)
	return nil
}

func main() {
	err := Newave(*x, *bcur, *qcur, *tf, *agg, *tf2, *agg2, tfrom, tto)
	if err != nil {
		log.Fatalf("Newave %s%s: %s", *bcur, *qcur, err)
	}
	if *promServer {
		<-make(chan int)
	}
}
