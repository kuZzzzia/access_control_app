package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/kuZzzzia/access_control_app/backend/api"
	"github.com/kuZzzzia/access_control_app/backend/specs"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog"

	"golang.org/x/sync/errgroup"
)

func Get(fileName string) (Config, error) {
	var cnf Config
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}
	err = yaml.Unmarshal(data, &cnf)
	if err != nil {
		return Config{}, err
	}
	return cnf, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.Ctx(ctx)

	var configPath string
	flag.StringVar(
		&configPath, "c", "config.yaml", "Used for set path to config file.")
	flag.Parse()

	cfg, err := Get(configPath)
	if err != nil {
		log.Fatal(err)
	}

	group := errgroup.Group{}

	group.Go(func() error {
		return StartHTTP(ctx, api.NewController(), &cfg)
	})

	signalListener := make(chan os.Signal, 1)
	defer close(signalListener)

	signal.Notify(signalListener,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	stop := <-signalListener
	logger.Info().Msg(fmt.Sprint("Received ", stop))
	logger.Info().Msg("Waiting for all jobs to stop")
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
