package db

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

type Waiter struct {
	QueueTypes []string
	Priority   int
	Ch         chan map[string]interface{}
}

type Queue struct {
	waiters []Waiter
	mu      *sync.RWMutex
	store   *DB
}

func NewQueue(ctx context.Context, store *DB) *Queue {
	q := Queue{
		waiters: make([]Waiter, 0),
		store:   store,
		mu:      new(sync.RWMutex),
	}

	go q.watch(ctx)

	return &q
}

func (q *Queue) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-q.store.WaitTask():
			for _, w := range q.waiters {
				if slices.Contains(w.QueueTypes, t.GetType()) && t.GetPriority() >= w.Priority {
					// Пробуем вытащить из базы
					task, err := q.store.Dequeue(ctx, w.QueueTypes, w.Priority)
					if err != nil {
						slog.ErrorContext(ctx, "dequeue error", slog.Any("error", err))
					}
					if task != nil {
						// Если получилось, то возвращаем
						w.Ch <- task
						break
					}
				}
			}
		}
	}
}

func (q *Queue) Dequeue(ctx context.Context, queueTypes []string, priority int) (map[string]interface{}, error) {
	// Пробуем вытащить из базы
	task, err := q.store.Dequeue(ctx, queueTypes, priority)
	if err != nil || task != nil {
		// Если получилось (или получили ошибку), то возвращаем
		return task, err
	}

	// Добавляем в список ожидания
	ch, c := q.Subscribe(queueTypes, priority)
	defer c()

	slog.DebugContext(ctx, "waiting for task")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case task = <-ch:
		return task, nil
	}
}

func (q *Queue) Subscribe(queueTypes []string, priority int) (<-chan map[string]interface{}, func()) {
	q.mu.Lock()
	defer q.mu.Unlock()

	ch := make(chan map[string]interface{}, 1)
	q.waiters = append(q.waiters, Waiter{
		QueueTypes: queueTypes,
		Priority:   priority,
		Ch:         ch,
	})

	return ch, func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		for i, channel := range q.waiters {
			if channel.Ch == ch {
				q.waiters = append(q.waiters[:i], q.waiters[i+1:]...)
				close(ch)
				break
			}
		}
	}
}
