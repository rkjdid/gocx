package chart

type Point struct {
	X, Y float64
}

func (p Point) Len() int {
	return 1
}

func (p Point) XY(i int) (float64, float64) {
	return p.X, p.Y
}
