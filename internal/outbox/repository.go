package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxRepository reads and updates outbox_events for the poller.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) FetchUnpublishedEvents(ctx context.Context, limit int) ([]Event, error) {
	query := `SELECT id, routing_key, payload
	          FROM outbox_events
	          WHERE NOT published
	          ORDER BY created_at
	          LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unpublished events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.RoutingKey, &e.Payload); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

func (r *OutboxRepository) MarkEventPublished(ctx context.Context, id int64) error {
	query := `UPDATE outbox_events SET published = TRUE WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark event published: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("outbox event %d not found", id)
	}

	return nil
}
