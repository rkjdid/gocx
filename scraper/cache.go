package scraper

import (
	"fmt"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	cacher *diskcache.Cache
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
	cacher = diskcache.New(cacheDir)
	Client = CacheClient(time.Hour * 24 * 10)
}

func CacheClient(maxAge time.Duration) *http.Client {
	if cacher == nil || maxAge == 0 {
		return http.DefaultClient
	}
	return &http.Client{
		Transport: Transport{
			t:      httpcache.NewTransport(cacher),
			maxAge: fmt.Sprintf("%.0f", maxAge.Seconds()),
		},
	}
}

type Transport struct {
	t      http.RoundTripper
	maxAge string
}

func (t Transport) RoundTrip(rq *http.Request) (*http.Response, error) {
	rq.Header.Set("Cache-Control", "max-age="+t.maxAge)
	return t.t.RoundTrip(rq)
}
