package cmd

import (
	"fmt"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	gocx "github.com/rkjdid/gocx/db"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

var (
	debug      bool
	db         *gocx.RedisDriver
	redisAddr  string
	promBind   string
	promHandle string
	promServer bool
	n          int

	rootCmd = &cobra.Command{
		Use:   "gocx",
		Short: "gocryptox",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			var err error
			// init db connection
			redisConn, err := redis.Dial("tcp", redisAddr)
			if err != nil {
				log.Fatalln("redis dial:", err)
			}
			db = &gocx.RedisDriver{Conn: redisConn}

			// init prometheus
			prometheus.MustRegister(sigCount, tradeCount)
			if promServer {
				http.Handle(promHandle, promhttp.Handler())
				fmt.Printf("%s%s\n", promBind, promHandle)
				go func() {
					log.Fatal("http listen:", http.ListenAndServe(promBind, nil))
				}()
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if promServer {
				fmt.Println("ctrl-c to quit")
				<-make(chan struct{})
			}
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&redisAddr, "redis", "localhost:6379", "redis server location")
	rootCmd.PersistentFlags().StringVar(&promBind, "prometheus-bind", ":8080", "prometheus bind")
	rootCmd.PersistentFlags().StringVar(&promHandle, "prometheus-handle", "/prometheus", "prometheus handle")
	rootCmd.PersistentFlags().BoolVar(&promServer, "prometheus-server", false, "enable prometheus webserver")
}
