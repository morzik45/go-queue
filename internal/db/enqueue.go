package db

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"log/slog"
	"strings"
	"time"
)

// Enqueue добавляет задачу в очередь
func (m *DB) Enqueue(ctx context.Context, qType string, priority int, data map[string]interface{}, reevaluation int) (string, error) {
	// Prepare the document to be inserted
	status := bson.M{
		"Status":    "Enqueued",
		"Timestamp": time.Now().UTC(),
	}
	if reevaluation > 0 {
		status["NextReevaluation"] = time.Now().UTC().Add(time.Duration(reevaluation) * time.Second)
	}
	doc := bson.M{
		"Statuses": bson.A{status},
		"Type":     qType,
		"Priority": priority,
		"Payload":  data,
	}

	// Insert the document into MongoDB
	res, err := m.queue.InsertOne(ctx, doc)
	if err != nil {
		slog.Error("failed to insert data into MongoDB",
			slog.String("operation", "enqueue"),
			slog.Any("error", err),
			slog.Any("document", doc),
		)
		return "", fmt.Errorf("failed to enqueue data: %w", err)
	}

	// Check the type of InsertedID
	oid, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		slog.ErrorContext(ctx, "Unexpected type for inserted ID",
			slog.String("operation", "enqueue"),
			slog.Any("error", err),
			slog.Any("insertedID", res.InsertedID),
		)
		return "", fmt.Errorf("unexpected type for inserted ID: %T", res.InsertedID)
	}

	m.enqChan <- &NewTask{
		Type:     qType,
		Priority: priority,
	}

	return oid.Hex(), nil
}

// Dequeue извлекает задачу из очереди
func (m *DB) Dequeue(ctx context.Context, qTypes []string, priority int) (map[string]interface{}, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	var result bson.M
	filter := bson.M{
		"Type":              bson.M{"$in": qTypes},
		"Priority":          bson.M{"$gte": priority},
		"Statuses.0.Status": "Enqueued",
		"$or": bson.A{
			bson.D{{"Statuses.0.NextReevaluation", bson.M{"$exists": false}}},
			bson.D{{"Statuses.0.NextReevaluation", bson.M{"$lt": time.Now().UTC()}}},
		},
	}

	opts := options.FindOneAndUpdate().SetSort(bson.D{
		{"Priority", -1},
		{"Statuses.0.Timestamp", 1},
	})

	status := bson.M{
		"Status":    "Processing",
		"Timestamp": time.Now().UTC(),
	}

	// TODO: Через какое время будем считать задачу невыполненной? получать это время с клиента?
	//if reevaluation > 0 {
	//	status["NextReevaluation"] = time.Now().UTC().Add(time.Duration(reevaluation) * time.Second)
	//}

	update := bson.M{
		"$push": bson.M{
			"Statuses": bson.M{
				"$each":     bson.A{status},
				"$position": 0,
			},
		},
	}

	cursor := m.queue.FindOneAndUpdate(ctx, filter, update, opts)
	if err := cursor.Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		slog.Error("failed to find data in mongodb", slog.Any("error", err))
		return nil, fmt.Errorf("failed to dequeue data: %w", err)
	}
	payloadBytes, err := bson.Marshal(result["Payload"])
	if err != nil {
		slog.Error("failed to marshal payload", slog.Any("error", err))
		return nil, fmt.Errorf("failed to dequeue data: %w", err)
	}

	var payload map[string]interface{}
	if err = bson.Unmarshal(payloadBytes, &payload); err != nil {
		slog.Error("failed to unmarshal payload", slog.Any("error", err))
		return nil, fmt.Errorf("failed to dequeue data: %w", err)
	}

	queueType, ok := result["Type"].(string)
	if !ok {
		slog.Error("failed to get queue type")
	}
	payload["queue_type"] = queueType

	id, ok := result["_id"].(bson.ObjectID)
	if !ok {
		slog.Error("failed to get id")
	}
	payload["id"] = id.Hex()

	return payload, nil
}

// Ack помечает задачу как выполненную
func (m *DB) Ack(ctx context.Context, id string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	oID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":               oID,
		"Statuses.0.Status": "Processing",
	}

	update := bson.M{
		"$push": bson.M{
			"Statuses": bson.M{
				"$each": bson.A{
					bson.M{
						"Status":    "Processed",
						"Timestamp": time.Now().UTC(),
					},
				},
				"$position": 0,
			},
		},
	}

	cursor := m.queue.FindOneAndUpdate(ctx, filter, update)
	if err := cursor.Err(); err != nil {
		slog.Error(
			"failed to find data in mongodb", slog.Any("error", err),
			slog.Any("filter", filter), slog.Any("update", update),
		)
		return err
	}
	return nil
}

// Failed помечает задачу как невыполненную
func (m *DB) Failed(ctx context.Context, id string, reevaluation int, message string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	oID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":               oID,
		"Statuses.0.Status": "Processing",
	}

	status := bson.M{
		"Status":    "Failed",
		"Timestamp": time.Now().UTC(),
		"Message":   message,
	}
	if reevaluation > 0 {
		status["NextReevaluation"] = time.Now().UTC().Add(time.Duration(reevaluation) * time.Second)
	}
	update := bson.M{
		"$push": bson.M{
			"Statuses": bson.M{
				"$each":     bson.A{status},
				"$position": 0,
			},
		}}

	cursor := m.queue.FindOneAndUpdate(ctx, filter, update)
	if err := cursor.Err(); err != nil {
		slog.Error("failed to find data in mongodb",
			slog.Any("error", err), slog.Any("filter", filter), slog.Any("update", update))
		return err
	}

	return nil
}

// Count возвращает количество задач в очереди для заданного типа очереди и по ключу данных
func (m *DB) Count(ctx context.Context, qType string, dataKey string, dataValue interface{}) (int64, error) {
	if ctx == nil {
		return 0, fmt.Errorf("context cannot be nil")
	}

	filter := bson.D{{"Statuses.0.Status", "Enqueued"}}

	if qType != "" {
		filter = append(filter, bson.E{Key: "Type", Value: qType})
	}

	if dataKey != "" {
		filter = append(filter, constructDataFilter(dataKey, dataValue))
	}

	count, err := m.queue.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("failed to count documents in mongodb", slog.Any("error", err), slog.Any("filter", filter))
		return 0, err
	}

	return count, nil
}

// Helper function to construct the data filter safely
func constructDataFilter(key string, value interface{}) bson.E {
	safeKey := sanitizeKey(key)
	return bson.E{Key: "Payload." + safeKey, Value: value}
}

// Helper function to sanitize keys to prevent injection attacks
func sanitizeKey(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}
