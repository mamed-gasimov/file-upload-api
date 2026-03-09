package outbox

import (
	"context"
	"log"
	"time"

	"github.com/mamed-gasimov/file-service/internal/messaging"
)

// Event mirrors the outbox_events row.
type Event struct {
	ID         int64
	RoutingKey string
	Payload    []byte
}

// Repository is the subset of persistence the poller needs.
type Repository interface {
	FetchUnpublishedEvents(ctx context.Context, limit int) ([]Event, error)
	MarkEventPublished(ctx context.Context, id int64) error
}

// Poller periodically reads unpublished outbox events and publishes them.
type Poller struct {
	repo      Repository
	publisher messaging.Publisher
	interval  time.Duration
	batchSize int
}

func NewPoller(repo Repository, pub messaging.Publisher, interval time.Duration, batchSize int) *Poller {
	return &Poller{
		repo:      repo,
		publisher: pub,
		interval:  interval,
		batchSize: batchSize,
	}
}

// Run blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	events, err := p.repo.FetchUnpublishedEvents(ctx, p.batchSize)
	if err != nil {
		log.Printf("outbox poller: fetch events: %v", err)
		return
	}

	for _, e := range events {
		if err := p.publisher.Publish(ctx, "", e.RoutingKey, e.Payload); err != nil {
			log.Printf("outbox poller: publish event %d: %v", e.ID, err)
			return // stop batch on first failure; retry next tick
		}

		if err := p.repo.MarkEventPublished(ctx, e.ID); err != nil {
			log.Printf("outbox poller: mark published %d: %v", e.ID, err)
			return
		}

		log.Printf("outbox poller: published event %d to %q", e.ID, e.RoutingKey)
	}
}
