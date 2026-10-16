package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const readBlockTimeout = 100 * time.Millisecond

type RedisQueue struct {
	client *redis.Client
	stream string
}

func NewRedisQueue(
	addr string,
	stream string,
) *RedisQueue {
	return &RedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
		stream: stream,
	}
}

type LeadCreatedEvent struct {
	LeadID      int64  `json:"lead_id"`
	RequestID   string `json:"request_id"`
	Traceparent string `json:"traceparent,omitempty"`
}

func (q *RedisQueue) PublishLeadCreated(
	ctx context.Context,
	leadID int64,
	requestID string,
) error {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	event := LeadCreatedEvent{
		LeadID:      leadID,
		RequestID:   requestID,
		Traceparent: carrier.Get("traceparent"),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal lead created event: %w", err)
	}

	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{
			"payload": string(data),
		},
	}).Err()
}

func (q *RedisQueue) ReadLeadCreated(
	ctx context.Context,
	count int,
	lastID string,
) ([]redis.XMessage, error) {
	if lastID == "" {
		lastID = "0"
	}

	result, err := q.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{q.stream, lastID},
		Count:   int64(count),
		Block:   readBlockTimeout,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, fmt.Errorf("read lead created events: %w", err)
	}

	var messages []redis.XMessage

	for _, stream := range result {
		messages = append(messages, stream.Messages...)
	}

	return messages, nil
}

func (q *RedisQueue) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}
