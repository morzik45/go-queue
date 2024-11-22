package db

import (
	"context"
	"errors"
	"github.com/morzik45/go-queue/pkg/utils"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"log/slog"
	"time"
)

type NewTask struct {
	Type     string
	Priority int
}

func (t *NewTask) GetType() string {
	return t.Type
}
func (t *NewTask) GetPriority() int {
	return t.Priority
}

type NewTaskI interface {
	GetType() string
	GetPriority() int
}

type DB struct {
	client  *mongo.Client
	db      *mongo.Database
	queue   *mongo.Collection
	enqChan chan NewTaskI
	Waiters *Queue
}

func NewMongoDB(ctx context.Context, cfg *viper.Viper) (m *DB, err error) {
	if cfg == nil || !cfg.IsSet("uri") || !cfg.IsSet("database") {
		return nil, errors.New("missing mongodb configuration")
	}
	m = &DB{
		enqChan: make(chan NewTaskI),
	}

	// Use the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(cfg.GetString("uri")).SetServerAPIOptions(serverAPI)

	m.client, err = mongo.Connect(opts)
	if err != nil {
		slog.Error("failed to connect to mongodb", slog.Any("error", err))
		return nil, err
	}

	ctxChild, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Убедимся что база на месте
	var result bson.M
	if err = m.client.Database("admin").RunCommand(ctxChild, bson.D{{"ping", 1}}).Decode(&result); err != nil {
		slog.Error("failed to ping mongodb", slog.Any("error", err))
	}

	go utils.CloseOnContext(ctx, m)

	m.db = m.client.Database(cfg.GetString("database"))

	m.queue = m.db.Collection("queue")

	// create dequeue index
	_, err = m.queue.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{"Type", 1},
			{"Priority", -1},
			{"Statuses.0.Status", 1},
			{"Statuses.0.NextReevaluation", 1},
			{"Statuses.0.Timestamp", 1},
		},
		Options: options.Index().SetName("dequeue_idx"),
	})
	if err != nil {
		slog.Warn("failed to create dequeue index", slog.Any("error", err))
		return nil, err
	}

	// create ttl index
	const ttl int32 = 60 * 60 * 24
	_, err = m.queue.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{"Statuses.0.Timestamp", 1}},
		Options: options.Index().SetExpireAfterSeconds(ttl),
	})
	if err != nil {
		slog.Warn("failed to create ttl index", slog.Any("error", err))
		return nil, err
	}

	slog.Info("connected to mongodb")

	m.Waiters = NewQueue(ctx, m)
	return m, nil
}

func (m *DB) WaitTask() <-chan NewTaskI {
	return m.enqChan
}

func (m *DB) Close() error {
	return m.client.Disconnect(context.Background())
}
