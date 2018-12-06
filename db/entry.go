package db

type Digester interface {
	Digest() (hash string, data []byte, err error)
}

type ZScorer interface {
	Digester

	// ZScore must return a score per day.
	ZScore() float64
}
