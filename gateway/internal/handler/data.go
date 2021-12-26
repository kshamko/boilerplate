package handler

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/kshamko/boilerplate/gateway/internal/models"
	"github.com/kshamko/boilerplate/gateway/internal/restapi/operations/data"
	"github.com/kshamko/boilerplate/grpc/pkg/grpcapi"
)

// Data struct to define Handle func on.
type Data struct {
	apiClient grpcapi.GRPCServiceClient
}

func NewData(apiClient grpcapi.GRPCServiceClient) *Data {
	return &Data{
		apiClient: apiClient,
	}
}

// Handle function processes http request, needed by swagger generated code.
func (dt *Data) Handle(in data.DataParams) middleware.Responder {

	ctx := context.Background()
	result, err := dt.apiClient.GetEntity(ctx, &grpcapi.GetReq{ID: in.ID})

	if err != nil {
		return data.NewDataNotFound().WithPayload(
			&models.APIInvalidResponse{
				Code:    404,
				Message: err.Error(),
			},
		)
	}

	return data.NewDataOK().WithPayload(
		&models.Data{
			ID:   result.ID,
			Name: result.Name,
		},
	)
}
