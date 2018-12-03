package db

type Digester interface {
	// Digest must return a unique id per data value.
	Digest() (id string, data []byte, err error)
}

type ZScore interface {
	// ZScore must return a score per day.
	ZScore() float64
}
