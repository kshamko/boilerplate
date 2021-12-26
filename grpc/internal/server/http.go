package server

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	gw "github.com/kshamko/boilerplate/grpc/pkg/grpcapi"
)

// ServeHTTP runs HTTP/REST gateway
func ServeHTTP(ctx context.Context, grpcAddr, httpAddr string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	if err := gw.RegisterGRPCServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	// graceful shutdown
	e := make(chan error, 1)
	go func() {
		e <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		return srv.Shutdown(ctx)
	case err := <-e:
		return err
	}
}
