package files

import "time"

type File struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Size               int64     `json:"size"`
	MimeType           string    `json:"mime_type"`
	ObjectKey          string    `json:"object_key"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Resume             *string   `json:"resume"`
	TranslationSummary *string   `json:"translation_summary"`
}

type OutboxEvent struct {
	ID         int64     `json:"id"`
	RoutingKey string    `json:"routing_key"`
	Payload    []byte    `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
	Published  bool      `json:"published"`
}
