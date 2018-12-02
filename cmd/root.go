package cmd

import (
	"fmt"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

var (
	debug      bool
	db         *redis.Client
	promBind   string
	promHandle string
	promServer bool
	n          int

	rootCmd = &cobra.Command{
		Use:   "gocx",
		Short: "gocryptox",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			var err error
			db, err = redis.Dial("tcp", "localhost:6379")
			if err != nil {
				log.Fatalln("redis dial:", err)
			}
			prometheus.MustRegister(sigCount, tradeCount)
			if promServer {
				http.Handle(promHandle, promhttp.Handler())
				fmt.Printf("localhost%s%s\n", promBind, promHandle)
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
	rootCmd.PersistentFlags().StringVar(&promBind, "prometheus-bind", ":8080", "prometheus bind")
	rootCmd.PersistentFlags().StringVar(&promHandle, "prometheus-handle", "/prometheus", "prometheus handle")
	rootCmd.PersistentFlags().BoolVar(&promServer, "prometheus-server", false, "enable prometheus webserver")
}
