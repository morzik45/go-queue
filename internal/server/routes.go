package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/morzik45/go-queue/internal/server/handlers"
	"github.com/spf13/viper"
)

func addRoutes(_ context.Context, mux *chi.Mux, cfg *viper.Viper, store *db.DB) {
	mux.Route("/api/v1", func(r chi.Router) {
		r.Post("/enqueue", handlers.Enqueue(store, cfg))
		r.Post("/dequeue", handlers.Dequeue(store, cfg))
		r.Post("/count", handlers.Count(store, cfg))
		r.Post("/ack", handlers.Ack(store, cfg))
		r.Post("/fail", handlers.Fail(store, cfg))
	})

	mux.Handle("/health", handlers.Health(store))
	mux.Handle("/", mux.NotFoundHandler())
}
