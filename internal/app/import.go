package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

type ObjectStore interface {
	Download(ctx context.Context, bucket, key string) ([]byte, error)
}

type CatalogStore interface {
	FindActiveSupplierID(ctx context.Context, code string) (uuid.UUID, error)
	UpsertOffers(ctx context.Context, supplierID uuid.UUID, offers []domain.CanonicalOffer) (domain.ImportStats, error)
}

type EventBus interface {
	PublishCompleted(ctx context.Context, event CompletedEvent) error
	PublishFailed(ctx context.Context, event FailedEvent) error
}

type WorkbookReader func(content []byte) (domain.Workbook, error)

type Service struct {
	Files   ObjectStore
	Catalog CatalogStore
	Events  EventBus
	Read    WorkbookReader
	Log     *slog.Logger
	Now     func() time.Time
}

func (s *Service) HandleFileReceived(ctx context.Context, event FileReceived) error {
	if s.Log == nil {
		s.Log = slog.Default()
	}
	if s.Now == nil {
		s.Now = func() time.Time { return time.Now().UTC() }
	}
	if !strings.EqualFold(strings.TrimSpace(event.SupplierCode), domain.SupplierCode) {
		s.Log.Info("ignoring supplier", "supplier_code", event.SupplierCode, "job_id", event.JobID)
		return nil
	}

	content, err := s.Files.Download(ctx, event.BlobContainer, event.BlobName)
	if err != nil {
		return s.fail(ctx, event, err)
	}

	wb, err := s.Read(content)
	if err != nil {
		return s.fail(ctx, event, err)
	}

	parsed, err := domain.ParseWorkbook(wb)
	if err != nil {
		return s.fail(ctx, event, err)
	}

	supplierID, err := s.Catalog.FindActiveSupplierID(ctx, domain.SupplierCode)
	if err != nil {
		return s.fail(ctx, event, err)
	}

	stats, err := s.Catalog.UpsertOffers(ctx, supplierID, parsed.Offers)
	if err != nil {
		return s.fail(ctx, event, err)
	}
	stats.RejectedRows = len(parsed.Stats.RejectedRows)

	if len(parsed.Stats.RejectedRows) > 0 {
		s.Log.Info("rejected rows", "job_id", event.JobID, "count", len(parsed.Stats.RejectedRows), "rows", parsed.Stats.RejectedRows)
	}

	return s.Events.PublishCompleted(ctx, CompletedEvent{
		OccurredAt:    s.Now(),
		CorrelationID: event.CorrelationID,
		JobID:         event.JobID,
		SupplierCode:  domain.SupplierCode,
		UploadedBy:    event.UploadedBy,
		Stats:         stats,
	})
}

func (s *Service) fail(ctx context.Context, event FileReceived, err error) error {
	var permanent *domain.PermanentError
	if !errors.As(err, &permanent) {
		return err
	}
	pubErr := s.Events.PublishFailed(ctx, FailedEvent{
		OccurredAt:    s.Now(),
		CorrelationID: event.CorrelationID,
		JobID:         event.JobID,
		SupplierCode:  domain.SupplierCode,
		UploadedBy:    event.UploadedBy,
		Stage:         permanent.Stage,
		ErrorCode:     permanent.Code,
		Message:       permanent.Message,
	})
	if pubErr != nil {
		return pubErr
	}
	return permanent
}
