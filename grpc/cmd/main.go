package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/InVisionApp/go-health"
	"github.com/jessevdk/go-flags"
	"github.com/kshamko/boilerplate/grpc/internal/datasource"
	"github.com/kshamko/boilerplate/grpc/internal/debug"
	"github.com/kshamko/boilerplate/grpc/internal/server"
	"golang.org/x/sync/errgroup"

	"github.com/sirupsen/logrus"
)

const APP_ID = "grpc"

func main() { //nolint: gocyclo
	var opts = struct {
		GRPCListen  string `long:"grpc.listen" env:"USER_GRPC_LISTEN" default:":8080" description:"GRPC server interface"`
		HTTPListen  string `long:"http.listen" env:"USER_HTTP_LISTEN" default:":8090" description:"http server interface"`
		DebugListen string `long:"debug.listen" env:"USER_DEBUG_LISTEN" default:":6060" description:"Interface for serve debug information(metrics/health/pprof)"`
		Verbose     bool   `long:"v" env:"USER_VERBOSE" description:"Enable Verbose log output"`
	}{}

	_, err := flags.Parse(&opts)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
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

	ds := datasource.NewMap()
	if err != nil {
		logger.Fatal(err)
	}

	gr.Go(func() error {
		dependencyHealth := &health.Config{
			Name:     "database",
			Checker:  ds,
			Interval: time.Duration(5) * time.Second,
			Fatal:    false,
		}
		healthd := health.New()
		healthd.AddCheck(dependencyHealth)

		d := debug.New(healthd)
		return d.Serve(appctx, opts.DebugListen)
	})

	gr.Go(func() error {
		return server.Serve(appctx, opts.GRPCListen, &stdGrpcLog{logger}, ds)
	})

	gr.Go(func() error {
		if opts.HTTPListen == "" {
			return nil
		}
		return server.ServeHTTP(appctx, opts.GRPCListen, opts.HTTPListen)
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
	if err != nil && err != errCanceled {
		logger.Fatal(err)
	}
}

type stdGrpcLog struct {
	*logrus.Entry
}

func (l *stdGrpcLog) V(level int) bool {
	return l.Logger.IsLevelEnabled(logrus.Level(level))
}
