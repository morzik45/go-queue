package handlers

import (
	"context"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/morzik45/go-queue/internal/logs"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"time"
)

type DequeueRequest struct {
	ApiKey     string   `json:"api_key"`
	QueueTypes []string `json:"queue_types"`
	Priority   int      `json:"priority,omitempty"`
	Timeout    int      `json:"timeout"`
}

func (dr DequeueRequest) Valid(_ context.Context) map[string]string {
	problems := make(map[string]string)
	if len(dr.QueueTypes) == 0 {
		problems["queue_types"] = "field queue_types is required"
	}

	return problems
}

type DequeueResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Problems map[string]string      `json:"problems"`
	Task     map[string]interface{} `json:"task"`
}

func Dequeue(store *db.DB, cfg *viper.Viper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := DequeueResponse{}
		req, problems, err := decodeValid[DequeueRequest](r)
		if err != nil {
			resp.Problems = problems
			resp.Message = err.Error()
			if err2 := encode(w, r, http.StatusBadRequest, resp); err2 != nil {
				slog.Error("ack send response error",
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

		if req.Timeout == 0 {
			req.Timeout = 10
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.Timeout)*time.Second)
		defer cancel()
		ctx = logs.WithValue(ctx, "queue_types", req.QueueTypes)
		ctx = logs.WithValue(ctx, "priority", req.Priority)

		task, err := store.Waiters.Dequeue(ctx, req.QueueTypes, req.Priority)
		var status int
		if err != nil {
			resp.Message = err.Error()
			status = http.StatusInternalServerError
			slog.ErrorContext(ctx, "dequeue error", slog.Any("error", err))
		} else if task != nil {
			resp.Success = true
			status = http.StatusOK
			resp.Task = task
			ctx = logs.WithValue(ctx, "task", task)
		} else {
			resp.Message = "no tasks"
			status = http.StatusNoContent
			slog.DebugContext(ctx, "no tasks")
		}
		if err = encode(w, r, status, resp); err != nil {
			slog.ErrorContext(ctx, "dequeue send response error", slog.Any("error", err))
		}
	}
}
