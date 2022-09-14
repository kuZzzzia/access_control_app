package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"access_control_app/backend/api"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zerolog.Ctx(ctx)
	if err != nil {
		log.Fatal(err)
	}

	group := errgroup.Group{}

	group.Go(func() error {
		return StartHTTP(ctx, api.NewController())
	})

	signalListener := make(chan os.Signal, 1)
	defer close(signalListener)

	signal.Notify(signalListener,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	stop := <-signalListener
	logger.Info("Received ", stop)
	logger.Info("Waiting for all jobs to stop")
}

func StartHTTP(ctx context.Context, ctrl *api.Controller) error {

	return nil
}
