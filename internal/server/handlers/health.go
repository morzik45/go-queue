package handlers

import (
	"github.com/morzik45/go-queue/internal/db"
	"net/http"
)

func Health(_ *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Ping DB and another checks
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}
