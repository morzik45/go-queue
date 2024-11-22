package handlers

import (
	"context"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
)

type AckRequest struct {
	ApiKey string `json:"api_key"`
	ID     string `json:"id"`
}

func (ar AckRequest) Valid(_ context.Context) map[string]string {
	problems := make(map[string]string)
	if ar.ID == "" {
		problems["id"] = "field id is required"
	}

	return problems
}

type AckResponse struct {
	Success  bool              `json:"success"`
	Message  string            `json:"message"`
	Problems map[string]string `json:"problems"`
}

func Ack(store *db.DB, cfg *viper.Viper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := AckResponse{}
		req, problems, err := decodeValid[AckRequest](r)
		if err != nil {
			resp.Problems = problems
			resp.Message = err.Error()
			if err2 := encode(w, r, http.StatusBadRequest, resp); err2 != nil {
				slog.Error("acl send response error",
					slog.Any("error", err2),
					slog.Any("problems", problems),
					slog.Any("first_error", err))
			}
			return
		}

		isAuth := checkApiKey(req.ApiKey, cfg)
		if !isAuth {
			resp.Message = "invalid api key"
			if err2 := encode(w, r, http.StatusUnauthorized, resp); err2 != nil {
				slog.Error("enqueue send response error",
					slog.Any("error", err2),
					slog.Any("problems", problems),
					slog.Any("first_error", err))
			}
			return
		}

		err = store.Ack(r.Context(), req.ID)
		var status int
		if err != nil {
			resp.Message = err.Error()
			status = http.StatusInternalServerError
		} else {
			resp.Success = true
			status = http.StatusOK
		}
		err = encode(w, r, status, resp)
		if err != nil {
			slog.Error("ack send response error", slog.Any("error", err))
		}
	}

}
