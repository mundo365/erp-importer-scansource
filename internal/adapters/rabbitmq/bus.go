package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/mundo365/erp-importer-scansource/internal/app"
	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

const (
	queueFileReceived = "import.file.received.v1"
	queueCompleted    = "import.completed.v1"
	queueFailed       = "import.failed.v1"
)

type fileReceivedJSON struct {
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID uuid.UUID `json:"correlation_id"`
	JobID         uuid.UUID `json:"job_id"`
	SupplierCode  string    `json:"supplier_code"`
	BlobContainer string    `json:"blob_container"`
	BlobName      string    `json:"blob_name"`
	ContentType   string    `json:"content_type"`
	UploadedBy    uuid.UUID `json:"uploaded_by"`
}

type completedJSON struct {
	EventType     string             `json:"event_type"`
	OccurredAt    time.Time          `json:"occurred_at"`
	CorrelationID uuid.UUID          `json:"correlation_id"`
	JobID         uuid.UUID          `json:"job_id"`
	SupplierCode  string             `json:"supplier_code"`
	UploadedBy    uuid.UUID          `json:"uploaded_by"`
	Stats         domain.ImportStats `json:"stats"`
}

type failedJSON struct {
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID uuid.UUID `json:"correlation_id"`
	JobID         uuid.UUID `json:"job_id"`
	SupplierCode  string    `json:"supplier_code"`
	UploadedBy    uuid.UUID `json:"uploaded_by"`
	Stage         string    `json:"stage"`
	ErrorCode     string    `json:"error_code"`
	Message       string    `json:"message"`
}

type Handler func(ctx context.Context, event app.FileReceived) error

type Bus struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	log  *slog.Logger
}

func NewBus(url string, log *slog.Logger) (*Bus, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	for _, q := range []string{queueFileReceived, queueCompleted, queueFailed} {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, fmt.Errorf("declare %s: %w", q, err)
		}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Bus{conn: conn, ch: ch, log: log}, nil
}

func (b *Bus) Close() error {
	if b.ch != nil {
		_ = b.ch.Close()
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

func (b *Bus) PublishCompleted(ctx context.Context, event app.CompletedEvent) error {
	payload := completedJSON{
		EventType:     queueCompleted,
		OccurredAt:    event.OccurredAt.UTC(),
		CorrelationID: event.CorrelationID,
		JobID:         event.JobID,
		SupplierCode:  event.SupplierCode,
		UploadedBy:    event.UploadedBy,
		Stats:         event.Stats,
	}
	if payload.OccurredAt.IsZero() {
		payload.OccurredAt = time.Now().UTC()
	}
	return b.publish(ctx, queueCompleted, event.CorrelationID, payload)
}

func (b *Bus) PublishFailed(ctx context.Context, event app.FailedEvent) error {
	payload := failedJSON{
		EventType:     queueFailed,
		OccurredAt:    event.OccurredAt.UTC(),
		CorrelationID: event.CorrelationID,
		JobID:         event.JobID,
		SupplierCode:  event.SupplierCode,
		UploadedBy:    event.UploadedBy,
		Stage:         event.Stage,
		ErrorCode:     event.ErrorCode,
		Message:       event.Message,
	}
	if payload.OccurredAt.IsZero() {
		payload.OccurredAt = time.Now().UTC()
	}
	return b.publish(ctx, queueFailed, event.CorrelationID, payload)
}

func (b *Bus) publish(ctx context.Context, queue string, correlationID uuid.UUID, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return b.ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		CorrelationId: correlationID.String(),
		Body:          body,
		Timestamp:     time.Now().UTC(),
	})
}

func (b *Bus) Consume(ctx context.Context, handler Handler) error {
	deliveries, err := b.ch.Consume(queueFileReceived, "erp-importer-scansource", false, false, false, false, nil)
	if err != nil {
		return err
	}
	b.log.Info("consuming", "queue", queueFileReceived)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("rabbitmq channel closed")
			}
			b.handleDelivery(ctx, delivery, handler)
		}
	}
}

func (b *Bus) handleDelivery(ctx context.Context, delivery amqp.Delivery, handler Handler) {
	var raw fileReceivedJSON
	if err := json.Unmarshal(delivery.Body, &raw); err != nil {
		b.log.Error("invalid message", "err", err)
		_ = delivery.Nack(false, false)
		return
	}
	event := app.FileReceived{
		EventType:     raw.EventType,
		OccurredAt:    raw.OccurredAt,
		CorrelationID: raw.CorrelationID,
		JobID:         raw.JobID,
		SupplierCode:  raw.SupplierCode,
		BlobContainer: raw.BlobContainer,
		BlobName:      raw.BlobName,
		ContentType:   raw.ContentType,
		UploadedBy:    raw.UploadedBy,
	}
	err := handler(ctx, event)
	if err == nil {
		_ = delivery.Ack(false)
		return
	}
	var permanent *domain.PermanentError
	if errors.As(err, &permanent) {
		b.log.Error("import failed", "job_id", event.JobID, "stage", permanent.Stage, "code", permanent.Code, "err", err)
		_ = delivery.Ack(false)
		return
	}
	b.log.Error("transient import error", "job_id", event.JobID, "err", err)
	_ = delivery.Nack(false, true)
}
