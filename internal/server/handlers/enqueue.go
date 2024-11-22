package handlers

import (
	"context"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
)

type EnqueueRequest struct {
	ApiKey       string                 `json:"api_key"`
	QueueType    string                 `json:"queue_type"`
	Priority     int                    `json:"priority,omitempty"`
	Payload      map[string]interface{} `json:"payload"`
	Reevaluation int                    `json:"reevaluation,omitempty"`
}

func (er EnqueueRequest) Valid(_ context.Context) map[string]string {
	problems := make(map[string]string)
	if er.QueueType == "" {
		problems["queue_type"] = "field queue_type is required"
	}
	if len(er.Payload) == 0 {
		problems["payload"] = "field payload is required"
	}
	return problems
}

type EnqueueResponse struct {
	Success  bool              `json:"success"`
	TaskID   string            `json:"task_id,omitempty"`
	Message  string            `json:"message,omitempty"`
	Problems map[string]string `json:"problems,omitempty"`
}

func Enqueue(store *db.DB, cfg *viper.Viper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := EnqueueResponse{}
		req, problems, err := decodeValid[EnqueueRequest](r)
		if err != nil {
			resp.Problems = problems
			resp.Message = err.Error()
			if err2 := encode(w, r, http.StatusBadRequest, resp); err2 != nil {
				slog.Error("enqueue send response error",
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

		var id string
		id, err = store.Enqueue(r.Context(), req.QueueType, req.Priority, req.Payload, req.Reevaluation)
		var status int
		if err != nil {
			resp.Message = err.Error()
			status = http.StatusInternalServerError
		} else {
			resp.Success = true
			resp.TaskID = id
			status = http.StatusCreated
		}
		err = encode(w, r, status, resp)
		if err != nil {
			slog.Error("enqueue send response error")
		}
	}
}
