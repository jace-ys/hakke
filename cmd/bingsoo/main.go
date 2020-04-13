package main

import (
	"context"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"golang.org/x/sync/errgroup"

	"github.com/jace-ys/bingsoo/pkg/bingsoo"
	"github.com/jace-ys/bingsoo/pkg/slack"
	"github.com/jace-ys/bingsoo/pkg/worker"
)

var logger log.Logger

func main() {
	c := parseCommand()

	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	slack := slack.NewHandler(c.slack.AccessToken, c.slack.SigningSecret)
	worker := worker.NewWorkerPool()
	bot := bingsoo.NewBingsooBot(logger, slack, worker)

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
	slack       slack.Config
}

func parseCommand() *config {
	var c config

	kingpin.Flag("port", "Port for the Bingsoo server.").Default("8080").IntVar(&c.port)
	kingpin.Flag("concurrency", "Number of concurrent workers to process tasks.").Default("4").IntVar(&c.concurrency)
	kingpin.Flag("slack-access-token", "Bot user access token for authenticating with the Slack API.").Required().StringVar(&c.slack.AccessToken)
	kingpin.Flag("slack-signing-secret", "Signing secret for verifying requests from Slack.").Required().StringVar(&c.slack.SigningSecret)
	kingpin.Parse()

	return &c
}

func exit(err error) {
	level.Error(logger).Log("event", "app.fatal", "msg", err)
	os.Exit(1)
}
