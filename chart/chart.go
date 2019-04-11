package chart

import (
	"fmt"
	"github.com/pplcc/plotext/custplotter"
	"github.com/rkjdid/gocx/ts"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"math"
	"time"
)

var (
	p        *plot.Plot
	plotters []plot.Plotter
	signals  []plot.Plotter

	candleWidth vg.Length
)

func init() {
	var err error
	p, err = plot.New()
	if err != nil {
		panic(err)
	}
}

func Plot() *plot.Plot {
	return p
}

func SetTitles(t, x, y string) {
	p.Title.Text = t
	p.X.Label.Text = x
	p.X.Padding = -1
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}
	p.Y.Label.Text = y
	p.Y.Padding = -1
	//p.Y.Scale = plot.LogScale{}
}

func SetRanges(minX, maxX, minY, maxY float64) {
	p.X.Min = math.Min(p.X.Min, minX)
	p.X.Max = math.Max(p.X.Max, maxX)
	p.Y.Min = math.Min(p.Y.Min, minY)
	p.Y.Max = math.Max(p.Y.Max, maxY)
}

func AddOHLCVs(data ts.OHLCVs) {
	if len(data) < 2 {
		panic(fmt.Errorf("too few data values in ohlcvs: %d", len(data)))
	}
	bars, _ := custplotter.NewCandlesticks(data)
	candleWidth = bars.CandleWidth
	plotters = append(plotters, bars)
	SetRanges(data.Range())
}

func AddLineWithStyle(xyer plotter.XYer, label string, style draw.LineStyle) {
	l, err := plotter.NewLine(xyer)
	if err != nil {
		panic(err)
	}
	l.LineStyle = style
	plotters = append(plotters, l)
	if label != "" {
		p.Legend.Add(label, l)
	}
}

func AddLines(labels []string, xyers ...plotter.XYer) {
	for i, xyer := range xyers {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		AddLine(xyer, label)
	}
}

func AddLine(xyer plotter.XYer, label string) {
	AddLineWithStyle(xyer, label, NextLineStyle())
}

func AddHorizontalFromToWithStyle(f float64, from, to float64, label string, style draw.LineStyle) {
	AddLineWithStyle(Horizontal{f, [2]float64{from, to}}, label, style)
}

func AddHorizontalFromTo(f float64, from, to float64, label string) {
	AddHorizontalFromToWithStyle(f, from, to, label, CurrentLineStyle())
}

func AddHorizontalFrom(f float64, from float64, label string) {
	AddHorizontalFromToWithStyle(f, from, p.X.Max, label, CurrentLineStyle())
}

func AddHorizontalWithStyle(f float64, label string, style draw.LineStyle) {
	AddHorizontalFromToWithStyle(f, p.X.Min, p.X.Max, label, style)
}

func AddHorizontal(f float64, label string) {
	AddHorizontalWithStyle(f, label, NextLineStyle())
}

func AddVertical(f float64, label string) {
	AddLineWithStyle(Vertical{f, [2]float64{p.Y.Min, p.Y.Max}}, label, GrayLine)
}

func AddSignal(t time.Time, buy bool, strong bool, y float64) {
	var style draw.GlyphStyle
	if buy {
		y = -y
		if strong {
			style = StrongBuy
		} else {
			style = Buy
		}
	} else {
		if strong {
			style = StrongSell
		} else {
			style = Sell
		}
	}
	AddSignalWithStyle(t, style, y)
}

func AddSignalWithStyle(t time.Time, style draw.GlyphStyle, y float64) {
	if t.Before(time.Now().Add(-time.Hour * 24 * 365 * 15)) {
		return
	}
	s, err := plotter.NewScatter(Point{float64(t.Unix()), y})
	if err != nil {
		panic(err)
	}
	s.GlyphStyle = style
	s.GlyphStyle.Radius *= W2
	signals = append(signals, s)
}

func Save(dimX, dimY float64, yGrid bool, file string) error {
	if yGrid {
		for _, v := range p.Y.Tick.Marker.Ticks(p.Y.Min, p.Y.Max) {
			l := plotter.DefaultLineStyle
			if !v.IsMinor() {
				l.Color = Color("e8e8e8")
			} else {
				l.Color = Color("c4c4c4")
			}
			AddHorizontalWithStyle(v.Value, "", l)
		}
	}
	for i := len(plotters) - 1; i >= 0; i-- {
		p.Add(plotters[i])
	}
	for _, s := range signals {
		p.Add(s)
	}
	return p.Save(vg.Length(dimX), vg.Length(dimY), file)
}
