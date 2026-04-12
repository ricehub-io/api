package repository

import (
	"context"
	"encoding/json"
)

type WebhookEventRepository struct {
	db DBExecutor
}

func NewWebhookEventRepository(db DBExecutor) *WebhookEventRepository {
	return &WebhookEventRepository{db}
}

func (r *WebhookEventRepository) InsertEvent(
	ctx context.Context,
	webhookID, eventType string,
	payload json.RawMessage,
) error {
	const query = `
	INSERT INTO webhook_events (polar_webhook_id, event_type, payload)
	VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, query, webhookID, eventType, payload)
	return err
}

func (r *WebhookEventRepository) EventProcessed(ctx context.Context, webhookID string) error {
	const query = `
	UPDATE webhook_events
	SET processed_at = now()
	WHERE polar_webhook_id = $1
	`
	_, err := r.db.Exec(ctx, query, webhookID)
	return err
}

func (r *WebhookEventRepository) SetEventError(ctx context.Context, webhookID, error string) error {
	const query = `
	UPDATE webhook_events
	SET error = $2
	WHERE polar_webhook_id = $1
	`
	_, err := r.db.Exec(ctx, query, webhookID, error)
	return err
}
