package server

import (
	"context"
	"net/http"
)

func initMiddlewares(ctx context.Context, handler http.Handler) http.Handler {

	return handler
}
