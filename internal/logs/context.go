package logs

import (
	"context"
	"log/slog"
	"sync"
)

type CtxHandler struct {
	handler slog.Handler
}

func NewCtxHandler(handler slog.Handler) slog.Handler {
	return CtxHandler{
		handler: handler,
	}
}

func (h CtxHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h CtxHandler) Handle(ctx context.Context, record slog.Record) error {
	if v, ok := ctx.Value(fields).(*sync.Map); ok {
		v.Range(func(key, val any) bool {
			if keyString, ok := key.(string); ok {
				record.AddAttrs(slog.Any(keyString, val))
			}
			return true
		})
	}
	return h.handler.Handle(ctx, record)
}

func (h CtxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return CtxHandler{h.handler.WithAttrs(attrs)}
}

func (h CtxHandler) WithGroup(name string) slog.Handler {
	return h.handler.WithGroup(name)
}

type contextKey string

var (
	fields contextKey = "slog_fields"
)

func WithValue(parent context.Context, key string, val any) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if v, ok := parent.Value(fields).(*sync.Map); ok {
		mapCopy := copySyncMap(v)
		mapCopy.Store(key, val)
		return context.WithValue(parent, fields, mapCopy)
	}
	v := &sync.Map{}
	v.Store(key, val)
	return context.WithValue(parent, fields, v)
}

func copySyncMap(m *sync.Map) *sync.Map {
	var cp sync.Map
	m.Range(func(k, v interface{}) bool {
		cp.Store(k, v)
		return true
	})
	return &cp
}
