package main

import (
	"context"
	"errors"

	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/kuZzzzia/access_control_app/backend/api"
	"github.com/kuZzzzia/access_control_app/specs"

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

type Config struct {
	Address  string `yaml:"address" validate:"required"`
	BasePath string `yaml:"base_path" validate:"required"`
}

func StartHTTP(ctx context.Context, ctrl *api.Controller, cfg *Config) error {
	router := chi.NewRouter()

	// for _, m := range middlewares {
	// 	router.Use(m)
	// }

	router.Handle("/", specs.HandlerFromMuxWithBaseURL(ctrl, router, cfg.BasePath))

	srv := http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}

	group := errgroup.Group{}
	group.Go(func() error {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	group.Go(func() error {
		<-ctx.Done()
		return srv.Shutdown(ctx)
	})

	return group.Wait()
}
