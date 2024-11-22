package handlers

import (
	"context"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
)

type CountRequest struct {
	ApiKey    string      `json:"api_key"`
	QueueType string      `json:"queue_type"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
}

func (cr CountRequest) Valid(_ context.Context) map[string]string {
	problems := make(map[string]string)
	if cr.QueueType == "" {
		problems["queue_type"] = "field queue_type is required"
	}
	if cr.Key == "" {
		problems["key"] = "field key is required"
	}
	if cr.Value == nil {
		problems["value"] = "field value is required"
	}

	return problems
}

type CountResponse struct {
	Success  bool              `json:"success"`
	Message  string            `json:"message,omitempty"`
	Problems map[string]string `json:"problems,omitempty"`
	Count    int               `json:"count"` // count of items in queue
}

func Count(store *db.DB, cfg *viper.Viper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := CountResponse{}
		req, problems, err := decodeValid[CountRequest](r)
		if err != nil {
			resp.Problems = problems
			resp.Message = err.Error()
			if err2 := encode(w, r, http.StatusBadRequest, resp); err2 != nil {
				slog.Error("count send response error",
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

		var count int64
		count, err = store.Count(r.Context(), req.QueueType, req.Key, req.Value)
		var status int
		if err != nil {
			resp.Message = err.Error()
			status = http.StatusInternalServerError
		} else {
			resp.Success = true
			resp.Count = int(count)
			status = http.StatusOK
		}
		err = encode(w, r, status, resp)
		if err != nil {
			slog.Error("count send response error", slog.Any("error", err))
		}
	}
}
