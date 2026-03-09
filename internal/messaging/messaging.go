package messaging

import "context"

type Publisher interface {
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	Close() error
}

// Delivery wraps a consumed message with acknowledgement controls.
type Delivery struct {
	Body []byte
	Ack  func() error
	Nack func(requeue bool) error
}

type Consumer interface {
	Consume(ctx context.Context, queue string) (<-chan Delivery, error)
	Close() error
}
