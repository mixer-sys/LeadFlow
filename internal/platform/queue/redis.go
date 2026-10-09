package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	stream string
}

func (q *RedisQueue) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func NewRedisQueue(addr, stream string) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &RedisQueue{
		client: client,
		stream: stream,
	}
}

type LeadCreatedEvent struct {
	LeadID int64 `json:"lead_id"`
}

func (q *RedisQueue) PublishLeadCreated(ctx context.Context, leadID int64) error {
	payload := LeadCreatedEvent{
		LeadID: leadID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal lead created event: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{
			"event": "lead.created",
			"data":  string(data),
		},
	}

	_, err = q.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("xadd: %w", err)
	}

	return nil
}

func (q *RedisQueue) ReadLeadCreated(ctx context.Context, count int64, lastID string) ([]redis.XMessage, error) {
	args := &redis.XReadArgs{
		Streams: []string{q.stream, lastID},
		Count:   count,
		Block:   2 * time.Second,
	}

	msgs, err := q.client.XRead(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("xread: %w", err)
	}

	if len(msgs) == 0 {
		return nil, nil
	}

	stream := msgs[0]
	return stream.Messages, nil
}

func (q *RedisQueue) AckLeadCreated(ctx context.Context, stream, id string) error {
	_, err := q.client.XAck(ctx, stream, id).Result()
	return err
}
