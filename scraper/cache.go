package scraper

import (
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	maxAge = "864000" // 10 day
)

func init() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Printf("scraper: cache disabled: os.UserCacheDir: %s", err)
		return
	}

	cacheDir = filepath.Join(cacheDir, "gocx")
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		log.Printf("scraper: cache disabled: os.UserCacheDir: %s", err)

	}
	c := diskcache.New(cacheDir)
	client = &http.Client{
		Transport: Transport{
			t:      httpcache.NewTransport(c),
			MaxAge: maxAge,
		},
	}
}

type Transport struct {
	t      http.RoundTripper
	MaxAge string
}

func (t Transport) RoundTrip(rq *http.Request) (*http.Response, error) {
	rq.Header.Set("Cache-Control", "max-age="+t.MaxAge)
	return t.t.RoundTrip(rq)
}
