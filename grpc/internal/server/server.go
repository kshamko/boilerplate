package server

import (
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/kshamko/boilerplate/grpc/internal/datasource"
	api "github.com/kshamko/boilerplate/grpc/pkg/grpcapi"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

// nolint
var (
	// ErrNotFound returns if no info was found
	ErrNotFound = status.Error(codes.NotFound, "not found")
	// ErrBadRequest returns if malformed input was provided
	ErrBadRequest = status.Error(codes.InvalidArgument, "invalid input")
)

//Datasource for invites database
type Datasource interface {
	GetEntry(ctx context.Context, id string) (datasource.Data, error)
	AddEntry(ctx context.Context, data datasource.Data) error
}

//Server for grpc server
type Server struct {
	db     Datasource
	logger grpclog.LoggerV2
}

//Serve creates and starts gRPC server
func Serve(ctx context.Context, listen string, l grpclog.LoggerV2, db Datasource) error {
	s := &Server{
		db:     db,
		logger: l,
	}

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		return errors.Wrap(err, "Coudn't start API server")
	}

	grpc_prometheus.EnableHandlingTimeHistogram(func(h *prometheus.HistogramOpts) {})

	mw := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
	}
	srv := grpc.NewServer(grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(mw...)))
	if l != nil {
		grpclog.SetLoggerV2(l)
	}
	api.RegisterGRPCServiceServer(srv, s)

	e := make(chan error, 1)
	go func() {
		e <- srv.Serve(lis)
	}()
	select {
	case <-ctx.Done():
		srv.GracefulStop()
		return nil
	case err := <-e:
		return err
	}
}

//GetUser return user by condition passed with user var
func (s *Server) GetEntity(ctx context.Context, r *api.GetReq) (*api.Entity, error) {
	if r == nil {
		return nil, ErrBadRequest
	}

	data, err := s.db.GetEntry(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return &api.Entity{
		ID:   data.ID,
		Name: data.Name,
	}, nil
}

//AddUser insert user into storage
func (s *Server) AddEntity(ctx context.Context, entity *api.Entity) (*api.Entity, error) {

	err := s.db.AddEntry(ctx, datasource.Data{ID: entity.ID, Name: entity.Name})
	return entity, err
}
