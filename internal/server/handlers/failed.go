package handlers

import (
	"context"
	"github.com/morzik45/go-queue/internal/db"
	"log/slog"
	"net/http"
)

type FailRequest struct {
	ID           string `json:"id"`
	Reevaluation int    `json:"reevaluation,omitempty"`
	Message      string `json:"message,omitempty"`
}

func (fr FailRequest) Valid(_ context.Context) map[string]string {
	problems := make(map[string]string)
	if fr.ID == "" {
		problems["id"] = "field id is required"
	}
	return problems
}

type FailResponse struct {
	Success  bool              `json:"success"`
	Message  string            `json:"message"`
	Problems map[string]string `json:"problems"`
}

func Fail(store *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := FailResponse{}
		req, problems, err := decodeValid[FailRequest](r)
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

		err = store.Failed(r.Context(), req.ID, req.Reevaluation, req.Message)
		var status int
		if err != nil {
			resp.Message = err.Error()
			status = http.StatusInternalServerError
		} else {
			resp.Success = true
			status = http.StatusOK
		}
		if err = encode(w, r, status, resp); err != nil {
			slog.Error("count send response error", slog.Any("error", err))
		}
		return
	}
}
