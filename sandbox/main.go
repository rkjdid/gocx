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
	"github.com/rkjdid/gocx/scraper"
	"github.com/rkjdid/gocx/scraper/binance"
	"github.com/rkjdid/gocx/strategy"
	"log"
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

	run        = flag.String("run", RunPrintTop, "bot action")
	n          = flag.Int("n", 10, "print/backtest top n markets")
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

const (
	RunBacktestTop = "backtestTop"
	RunBacktestOne = "backtestOne"
	RunPrintTop    = "printTop"
)

func runHelpText() string {
	return fmt.Sprintf("bot action: %v",
		[]string{
			RunBacktestTop,
			RunBacktestOne,
			RunPrintTop,
		})
}

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
	if *qcur == "" {
		*qcur = "BTC"
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

var (
	macdSlow, macdFast strategy.MACDOpts
)

func main() {
	ratio := 4
	macdSlow = strategy.MACDOpts{12, 26, 9}
	macdFast = strategy.MACDOpts{ratio * 12, ratio * 26, 9}

	switch *run {
	case RunBacktestOne:
		BacktestOne(*bcur, *qcur)
	case RunBacktestTop:
		BacktestTop(*n)
	case RunPrintTop:
		PrintBinanceTop(*n)
	default:
		s, _ := os.Executable()
		log.Fatalf("unsuported command: %s. <%s> -h", *run, s)
	}

	if *promServer {
		<-make(chan int)
	}
}

func BacktestOne(bcur, qcur string) {
	res, err := Newave(*x, bcur, qcur, *tf, *agg, macdSlow, macdFast, tfrom, tto)
	if err != nil {
		log.Printf("Newave %s:%s%s - %s", *x, bcur, qcur, err)
		return
	}
	for _, p := range res.Positions {
		fmt.Println(p)
	}
	fmt.Println(res)
}

func PrintBinanceTop(n int) {
	tickers, err := binance.FetchTopTickers("", "BTC")
	if err != nil {
		log.Fatal(err)
	}
	if n <= 0 {
		n = len(tickers)
	}
	for _, t := range tickers[:n] {
		fmt.Println(t.Symbol, t.QuoteVolume)
	}
}

func BacktestTop(n int) {
	tickers, err := binance.FetchTopTickers("", "BTC")
	if err != nil {
		log.Fatal(err)
	}
	if n <= 0 {
		n = len(tickers)
	}
	for _, v := range tickers[:n] {
		BacktestOne(v.Base, v.Quote)
	}
}
