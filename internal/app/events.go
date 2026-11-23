package app

import (
	"time"

	"github.com/google/uuid"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

type FileReceived struct {
	EventType     string
	OccurredAt    time.Time
	CorrelationID uuid.UUID
	JobID         uuid.UUID
	SupplierCode  string
	BlobContainer string
	BlobName      string
	ContentType   string
	UploadedBy    uuid.UUID
}

type CompletedEvent struct {
	OccurredAt    time.Time
	CorrelationID uuid.UUID
	JobID         uuid.UUID
	SupplierCode  string
	UploadedBy    uuid.UUID
	Stats         domain.ImportStats
}

type FailedEvent struct {
	OccurredAt    time.Time
	CorrelationID uuid.UUID
	JobID         uuid.UUID
	SupplierCode  string
	UploadedBy    uuid.UUID
	Stage         string
	ErrorCode     string
	Message       string
}
