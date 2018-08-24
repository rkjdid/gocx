package chart

import (
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"image/color"
	"strconv"
	"strings"
	"sync"
)

const (
	W1               = vg.Length(1)
	W2               = vg.Length(2)
	W4               = vg.Length(4)
	colorThemeModulo = 5
)

var (
	lineWidth = W2
	lineColor = 0
	lineLock  = sync.Mutex{}

	Colors = []color.Color{
		// POP https://coolors.co/50514f-f25f5c-ffe066-247ba0-70c1b3
		Color("50514F"),
		Color("F25F5C"),
		Color("FFE066"),
		Color("247BA0"),
		Color("70C1B3"),

		// COOL https://coolors.co/d8dbe2-a9bcd0-58a4b0-373f51-1b1b1e
		Color("D8DBE2"),
		Color("A9BCD0"),
		Color("58A4B0"),
		Color("373F51"),
		Color("1B1B1E"),
	}

	Red   = Color("ff3300")
	Green = Color("339933")
)

func Color(hash string) color.Color {
	if strings.HasPrefix(hash, "#") {
		hash = hash[1:]
	}
	if len(hash) != 6 && len(hash) != 8 {
		return color.Black
	}
	var c color.RGBA
	c.A = 255
	cs := []*uint8{&c.R, &c.G, &c.B, &c.A}
	for i := 0; i < len(hash); i += 2 {
		ui, err := strconv.ParseUint(hash[i:i+2], 16, 8)
		if err != nil {
			return c
		}
		*cs[i/2] = uint8(ui)
	}
	return c
}

func SetLineWidth(length vg.Length) {
	lineLock.Lock()
	lineWidth = length
	lineLock.Unlock()
}

func CurrentLineStyle() draw.LineStyle {
	return draw.LineStyle{
		Color: Colors[lineColor],
		Width: lineWidth,
	}
}

func NextLineStyle() draw.LineStyle {
	lineLock.Lock()
	d := draw.LineStyle{
		Color: Colors[lineColor],
		Width: lineWidth,
	}
	lineColor = (lineColor + 1) % len(Colors)
	lineLock.Unlock()
	return d
}

func ResetLineColor() {
	lineLock.Lock()
	lineColor = -1
	lineLock.Unlock()
}

func NextLineTheme() {
	lineLock.Lock()
	lineColor = (int(lineColor/colorThemeModulo) + 1) % (len(Colors) / colorThemeModulo)
	lineColor *= colorThemeModulo
	lineLock.Unlock()
}
