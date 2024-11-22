package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/spf13/viper"
	"net/http"
)

func NewServer(ctx context.Context, config *viper.Viper, store *db.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	addRoutes(
		ctx,
		r,
		config,
		store,
	)

	return r
}
