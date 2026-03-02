package outbound

import (
	"context"

	"github.com/nexlified/dam/internal/domain"
)

// EventPublisherPort sends domain events to a queue or stream.
type EventPublisherPort interface {
	Publish(ctx context.Context, event *domain.Event) error
}
