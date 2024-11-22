package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func encode[T any](w http.ResponseWriter, _ *http.Request, status int, v T) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

// Validator - это объект, который можно валидировать.
type Validator interface {
	// Valid проверяет объект и возвращает любые
	// проблемы. Если len(problems) == 0, тогда
	// объект валиден.
	Valid(ctx context.Context) (problems map[string]string)
}

func decodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}
