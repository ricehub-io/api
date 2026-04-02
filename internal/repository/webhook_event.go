package repository

import (
	"context"
	"encoding/json"
)

func InsertEvent(webhookID, eventType string, payload json.RawMessage) error {
	const query = `
	INSERT INTO webhook_events (polar_webhook_id, event_type, payload)
	VALUES ($1, $2, $3)
	`
	_, err := db.Exec(context.Background(), query, webhookID, eventType, payload)
	return err
}

func EventProcessed(webhookID string) error {
	const query = `
	UPDATE webhook_events
	SET processed_at = now()
	WHERE polar_webhook_id = $1
	`
	_, err := db.Exec(context.Background(), query, webhookID)
	return err
}

func SetEventError(webhookID, error string) error {
	const query = `
	UPDATE webhook_events
	SET error = $2
	WHERE polar_webhook_id = $1
	`
	_, err := db.Exec(context.Background(), query, webhookID, error)
	return err
}
