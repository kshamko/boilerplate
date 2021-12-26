package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/InVisionApp/go-health"
	"github.com/go-openapi/loads"
	flags "github.com/jessevdk/go-flags"
	"github.com/kshamko/boilerplate/gateway/internal/datasource"
	"github.com/kshamko/boilerplate/gateway/internal/debug"
	"github.com/kshamko/boilerplate/gateway/internal/handler"
	"github.com/kshamko/boilerplate/gateway/internal/restapi"
	"github.com/kshamko/boilerplate/gateway/internal/restapi/operations"
	"github.com/kshamko/boilerplate/grpc/pkg/grpcapi"
	"golang.org/x/sync/errgroup"
)

const APP_ID = "gateway"

func main() { //nolint: funlen
	var opts = struct {
		HTTPListenHost string `long:"http.listen.host" env:"HTTP_LISTEN_HOST" default:"" description:"http server interface host"`
		HTTPListenPort int    `long:"http.listen.port" env:"HTTP_LISTEN_PORT" default:"8081" description:"http server interface port"`
		DebugListen    string `long:"debug.listen" env:"DEBUG_LISTEN" default:":6060" description:"Interface for serve debug information(metrics/health/pprof)"`
		Verbose        bool   `long:"v" env:"VERBOSE" description:"Enable Verbose log output"`
		GRPCEnpoint    string `long:"service.grpc" env:"SERVICE_GRPC" default:":8081"`
	}{}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	logger := logrus.WithField("app_id", APP_ID)
	logger.Logger.SetOutput(os.Stdout)
	logger.Logger.SetLevel(logrus.InfoLevel)

	if opts.Verbose {
		logger.Logger.SetLevel(logrus.DebugLevel)
	}

	logger.Infof("Launching Application with: %+v", opts)

	gr, appctx := errgroup.WithContext(context.Background())
	gr.Go(func() error {
		healthd := health.New()
		d := debug.New(healthd)

		return d.Serve(appctx, opts.DebugListen)
	})

	grpcapi.NewClient()

	gr.Go(func() error {
		swaggerSpec, err := loads.Embedded(restapi.SwaggerJSON, restapi.FlatSwaggerJSON)
		if err != nil {
			logger.Error(err)
			return err
		}
		api := operations.NewBoilerPlateGWSwaggerAPI(swaggerSpec)
		api.DataDataHandler = handler.NewData(
			datasource.NewMap(),
		)

		server := restapi.NewServer(api)
		// nolint: errcheck
		defer server.Shutdown()

		server.Host = opts.HTTPListenHost
		server.Port = opts.HTTPListenPort

		return server.Serve()
	})

	errCanceled := errors.New("Canceled")
	gr.Go(func() error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		cusr := make(chan os.Signal, 1)
		signal.Notify(cusr, syscall.SIGUSR1)
		for {
			select {
			case <-appctx.Done():
				return nil
			case <-sigs:
				logger.Info("Caught stop signal. Exiting ...")
				return errCanceled
			case <-cusr:
				if logger.Level == logrus.DebugLevel {
					logger.Logger.SetLevel(logrus.InfoLevel)
					logger.Info("[INFO] Caught SIGUSR1 signal. Log level changed to INFO")
					continue
				}
				logger.Info("Caught SIGUSR1 signal. Log level changed to DEBUG")
				logger.Logger.SetLevel(logrus.DebugLevel)
			}
		}
	})

	err = gr.Wait()
	if err != nil && errors.Is(err, errCanceled) {
		log.Fatal(err)
	}
}

func initGRPCClient()
