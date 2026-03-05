package redis

import (
	"context"
	"encoding/json"

	"github.com/nexlified/dam/domain"
	goredis "github.com/redis/go-redis/v9"
)

const streamName = "dam:events"

type EventPublisher struct {
	client *Client
}

func NewEventPublisher(client *Client) *EventPublisher {
	return &EventPublisher{client: client}
}

func (p *EventPublisher) Publish(ctx context.Context, event *domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.client.Raw().XAdd(ctx, &goredis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"type":    string(event.Type),
			"org_id":  event.OrgID.String(),
			"payload": string(payload),
		},
	}).Err()
}
