package cmd

import (
	"fmt"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_db "github.com/rkjdid/gocx/db"
	"github.com/rkjdid/gocx/scraper"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

var (
	debug      bool
	db         *_db.RedisDriver
	redisAddr  string
	promBind   string
	promHandle string
	promServer bool
	n          int
	zkey       string

	rootCmd = &cobra.Command{
		Use:   "gocx",
		Short: "gocryptox",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// propagate debug flag
			scraper.Debug = debug

			// init db connection
			//redisConn, err := redis.Dial("tcp", redisAddr)
			//if err != nil {
			//	log.Fatalln("redis dial:", err)
			//}
			p, err := pool.New("tcp", redisAddr, 12)
			if err != nil {
				log.Fatalln("redis pool:", err)
			}
			db = &_db.RedisDriver{Pool: p}

			// init prometheus
			prometheus.MustRegister(signals, trades)
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

	// prometheus metrics
	signals = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signal", Help: "signals",
	}, []string{"name", "action"})

	trades = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trade", Help: "trades",
	}, []string{"direction", "quantity", "price"})
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
	rootCmd.PersistentFlags().StringVar(&zkey, "zkey", "backtest", "redis key for sorted set")

	rootCmd.AddCommand(backtestCmd, topCmd, showCmd, optimizeCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use: "flushdb",
		Run: func(cmd *cobra.Command, args []string) {
			err := db.Pool.Cmd("flushdb").Err
			if err != nil {
				log.Fatalln("flushdb", err)
			}
		},
	})
}

// TraverseRunHooks modifies c's PersistentPreRun* and PersistentPostRun*
// functions (when present) so that they will search c's command chain and
// invoke the corresponding hook of the first parent that provides a hook.
// When used on every command in the chain the invocation of hooks will be
// propagated up the chain to the root command.
//
// In the case of PersistentPreRun* hooks the parent hook is invoked before the
// child hook.  In the case of PersistentPostRun* the child hook is invoked
// first.
//
// Use it in place of &cobra.Command{}, e.g.
//     command := TraverseRunHooks(&cobra.Command{
//     	PersistentPreRun: func ...,
//     })
func TraverseRunHooks(c *cobra.Command) *cobra.Command {
	preRunE := c.PersistentPreRunE
	preRun := c.PersistentPreRun
	if preRunE != nil || preRun != nil {
		c.PersistentPreRun = nil
		c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			for p := c.Parent(); p != nil; p = p.Parent() {
				if p.PersistentPreRunE != nil {
					if err := p.PersistentPreRunE(cmd, args); err != nil {
						return err
					}
					break
				} else if p.PersistentPreRun != nil {
					p.PersistentPreRun(cmd, args)
					break
				}
			}

			if preRunE != nil {
				return preRunE(cmd, args)
			}

			preRun(cmd, args)

			return nil
		}
	}

	postRunE := c.PersistentPostRunE
	postRun := c.PersistentPostRun
	if postRunE != nil || postRun != nil {
		c.PersistentPostRun = nil
		c.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
			if postRunE != nil {
				if err := postRunE(cmd, args); err != nil {
					return err
				}
			} else if postRun != nil {
				postRun(cmd, args)
			}

			for p := c.Parent(); p != nil; p = p.Parent() {
				if p.PersistentPostRunE != nil {
					if err := p.PersistentPostRunE(cmd, args); err != nil {
						return err
					}
					break
				} else if p.PersistentPostRun != nil {
					p.PersistentPostRun(cmd, args)
					break
				}
			}

			return nil
		}
	}

	return c
}
