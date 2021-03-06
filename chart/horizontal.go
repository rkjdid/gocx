package chart

type Horizontal struct {
	Y float64
	X [2]float64
}

func (Horizontal) Len() int {
	return 2
}

func (h Horizontal) XY(i int) (float64, float64) {
	return h.X[i], h.Y
}

type Vertical struct {
	X float64
	Y [2]float64
}

func (Vertical) Len() int {
	return 2
}

func (v Vertical) XY(i int) (float64, float64) {
	return v.X, v.Y[i]
}
