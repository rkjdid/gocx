package db

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Digester interface {
	Digest() (hash string, data []byte, err error)
}

type ZScorer interface {
	Digester

	// ZScore must return a score per day.
	ZScore() float64
}

func JSONDigest(prefix string, v interface{}) (hash string, data []byte, err error) {
	var b bytes.Buffer
	err = json.NewEncoder(&b).Encode(v)
	if err != nil {
		return
	}
	data = b.Bytes()
	shasum := sha256.Sum256(data)
	if len(prefix) > 0 {
		prefix = prefix + ":"
	}
	hash = fmt.Sprintf("%s%x", prefix, shasum[:10])
	//fmt.Printf("%s\n", data)
	return
}

type TopZScorer []ZScorer

func (s TopZScorer) Less(n, m int) bool {
	return s[m].ZScore() < s[n].ZScore()
}

func (s TopZScorer) Len() int {
	return len(s)
}

func (s TopZScorer) Swap(n, m int) {
	s[n], s[m] = s[m], s[n]
}
