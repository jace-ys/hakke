package main

import (
	"context"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/go-kit/kit/log"
	"golang.org/x/sync/errgroup"

	"github.com/jace-ys/bingsoo/pkg/bingsoo"
	"github.com/jace-ys/bingsoo/pkg/postgres"
	"github.com/jace-ys/bingsoo/pkg/redis"
)

var logger log.Logger

func main() {
	c := parseCommand()

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	postgres, err := postgres.NewClient(c.database.Host, c.database.User, c.database.Password, c.database.Database)
	if err != nil {
		exit(err)
	}
	redis, err := redis.NewClient(c.cache.Host)
	if err != nil {
		exit(err)
	}

	bot := bingsoo.NewBingsooBot(logger, postgres, redis, c.bot.SigningSecret)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return bot.StartServer(c.port)
	})
	g.Go(func() error {
		return bot.StartWorkers(ctx, c.concurrency)
	})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			if err := bot.Shutdown(ctx); err != nil {
				return err
			}
			return ctx.Err()
		}
	})

	if err := g.Wait(); err != nil {
		exit(err)
	}
}

type config struct {
	port        int
	concurrency int
	bot         bingsoo.BingsooBotConfig
	database    postgres.ClientConfig
	cache       redis.ClientConfig
}

func parseCommand() *config {
	var c config

	kingpin.Flag("port", "Port for the Bingsoo server.").Envar("PORT").Default("8080").IntVar(&c.port)
	kingpin.Flag("concurrency", "Number of concurrent workers to process tasks.").Envar("CONCURRENCY").Default("4").IntVar(&c.concurrency)
	kingpin.Flag("signing-secret", "Signing secret for verifying requests from Slack.").Envar("SIGNING_SECRET").Required().StringVar(&c.bot.SigningSecret)
	kingpin.Flag("postgres-host", "Host for connecting to Postgres").Envar("POSTGRES_HOST").Default("127.0.0.1:5432").StringVar(&c.database.Host)
	kingpin.Flag("postgres-user", "User for connecting to Postgres").Envar("POSTGRES_USER").Default("postgres").StringVar(&c.database.User)
	kingpin.Flag("postgres-password", "Password for connecting to Postgres").Envar("POSTGRES_PASSWORD").Required().StringVar(&c.database.Password)
	kingpin.Flag("postgres-db", "Database for connecting to Postgres").Envar("POSTGRES_DB").Default("postgres").StringVar(&c.database.Database)
	kingpin.Flag("redis-host", "Host for connecting to Redis").Envar("REDIS_HOST").Default("127.0.0.1:6379").StringVar(&c.cache.Host)
	kingpin.Parse()

	return &c
}

func exit(err error) {
	logger.Log("event", "app.fatal", "error", err)
	os.Exit(1)
}
