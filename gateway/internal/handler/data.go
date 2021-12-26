package handler

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/kshamko/boilerplate/gateway/internal/datasource"
	"github.com/kshamko/boilerplate/gateway/internal/models"
	"github.com/kshamko/boilerplate/gateway/internal/restapi/operations/data"
)

// RoutesDatasource interface for routing data.
// Needed in case we want to mock it and cover handler with tests.
type Datasource interface {
	GetDataByID(ctx context.Context, id string) (datasource.Data, error)
}

// Routes struct to define Handle func on.
type Data struct {
	ds Datasource
}

// NewRoutes return Routes object.
func NewData(ds Datasource) *Data {
	return &Data{
		ds: ds,
	}
}

// Handle function processes http request, needed by swagger generated code.
func (dt *Data) Handle(in data.DataParams) middleware.Responder {

	ctx := context.Background()
	result, err := dt.ds.GetDataByID(ctx, in.ID)

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
