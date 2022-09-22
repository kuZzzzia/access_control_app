package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/units"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/kuZzzzia/access_control_app/backend/api"
	"github.com/kuZzzzia/access_control_app/backend/service"
	"github.com/kuZzzzia/access_control_app/backend/specs"
	"github.com/kuZzzzia/access_control_app/backend/storage/postgres"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog/log"

	"golang.org/x/sync/errgroup"

	storage "github.com/kuZzzzia/access_control_app/backend/storage/minio"
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

	var configPath string
	flag.StringVar(
		&configPath, "c", "config.yaml", "Used for set path to config file.")
	flag.Parse()

	cfg, err := Get(configPath)
	if err != nil {
		log.Fatal().Err(err)
	}

	var limit int64
	if cfg.SizeLimitSrt == "" {
		cfg.SizeLimitSrt = "10GB"
	}
	limit, parseLimitErr := units.ParseStrictBytes(cfg.SizeLimitSrt)
	if parseLimitErr != nil {
		log.Fatal().Err(parseLimitErr).Msg("convert limit to bytes")
	}

	db, err := CreateConnect(cfg.Connection)
	if err != nil {
		log.Fatal().Err(err).Msg("failed opening connection to sqlite")
	}

	minioClient, errMinio := minio.New(cfg.Minio.Address, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKeyID, cfg.Minio.SecretAccessKey, ""),
		Secure: cfg.Minio.SSL,
	})
	if errMinio != nil {
		log.Fatal().Err(errMinio).Str("address", cfg.Minio.Address).Msg("create minioClient")
	}

	repo := postgres.NewRepository(db)

	apiServer := api.NewController(
		service.NewObjectService(
			repo,
			&storage.ObjectStorage{
				Minio:  minioClient,
				Region: cfg.Minio.Region,
				Bucket: cfg.Minio.BucketID,
			},
		),
		repo, cfg.DenyTypes, limit)

	group := errgroup.Group{}

	group.Go(func() error {
		return StartHTTP(ctx, apiServer, &cfg)
	})

	signalListener := make(chan os.Signal, 1)
	defer close(signalListener)

	signal.Notify(signalListener,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	stop := <-signalListener
	log.Info().Msg(fmt.Sprint("Received ", stop))
	log.Info().Msg("Waiting for all jobs to stop")
}

type Config struct {
	Address  string `yaml:"address" validate:"required"`
	BasePath string `yaml:"base_path" validate:"required"`

	Connection string        `yaml:"postgresql"`
	Minio      storage.Minio `yaml:"minio"`

	SizeLimitSrt string            `yaml:"size_limit"`
	DenyTypes    map[string]string `yaml:"deny_types"`
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

func CreateConnect(connection string) (*sql.DB, error) {
	var (
		ctor driver.Connector
	)

	drv := stdlib.GetDefaultDriver().(*stdlib.Driver)

	ctor, err := drv.OpenConnector(connection)
	if err != nil {
		return nil, err
	}

	// TODO:

	db := sql.OpenDB(ctor)

	return db, nil
}
