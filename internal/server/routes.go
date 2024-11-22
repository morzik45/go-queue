package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/morzik45/go-queue/internal/server/handlers"
	"github.com/spf13/viper"
)

func addRoutes(_ context.Context, mux *chi.Mux, cfg *viper.Viper, store *db.DB) {
	mux.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.BasicAuth("DeepBotQueue", cfg.GetStringMapString("auth")))

		r.Post("/enqueue", handlers.Enqueue(store))
		r.Post("/dequeue", handlers.Dequeue(store))
		r.Post("/count", handlers.Count(store))
		r.Post("/ack", handlers.Ack(store))
		r.Post("/fail", handlers.Fail(store))
	})

	mux.Handle("/health", handlers.Health(store))
	mux.Handle("/", mux.NotFoundHandler())
}
