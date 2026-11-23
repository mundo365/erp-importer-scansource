package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

type memFiles struct {
	objects map[string][]byte
	err     error
}

func (m *memFiles) Download(_ context.Context, bucket, key string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	content, ok := m.objects[bucket+"/"+key]
	if !ok {
		return nil, domain.Permanent(domain.StageParse, "blob_not_found", "objeto nao encontrado")
	}
	return content, nil
}

type memCatalog struct {
	supplierID uuid.UUID
	supplierOK bool
	offers     []domain.CanonicalOffer
	failUpsert error
}

func (m *memCatalog) FindActiveSupplierID(_ context.Context, code string) (uuid.UUID, error) {
	if !m.supplierOK || code != domain.SupplierCode {
		return uuid.Nil, domain.Permanent(domain.StageUpsert, "supplier_not_found", "fornecedor scansource ativo nao encontrado")
	}
	return m.supplierID, nil
}

func (m *memCatalog) UpsertOffers(_ context.Context, _ uuid.UUID, offers []domain.CanonicalOffer) (domain.ImportStats, error) {
	if m.failUpsert != nil {
		return domain.ImportStats{}, m.failUpsert
	}
	m.offers = append([]domain.CanonicalOffer(nil), offers...)
	return domain.ImportStats{
		OffersUpserted:       len(offers),
		ProductsCreatedDraft: len(offers),
		ProductsExisting:     0,
	}, nil
}

type memBus struct {
	completed []CompletedEvent
	failed    []FailedEvent
}

func (m *memBus) PublishCompleted(_ context.Context, event CompletedEvent) error {
	m.completed = append(m.completed, event)
	return nil
}

func (m *memBus) PublishFailed(_ context.Context, event FailedEvent) error {
	m.failed = append(m.failed, event)
	return nil
}

func sampleWorkbook() domain.Workbook {
	return domain.Workbook{
		"CSP": {
			{},
			{"Sku", "Offer Display Name", "Preço Unitário", "Tempo de contrato", "Segmento", "Categoria", "Ciclo de pagamento", "Billing Plan"},
			{"CFQ7TTC0HL8Z-1-P1Y-Annual-BR", "Audit Log", "143.12", "Anual", "Commercial", "x", "Anual", ""},
		},
	}
}

func TestHandleFileReceivedSuccess(t *testing.T) {
	supplierID := uuid.New()
	jobID := uuid.New()
	correlation := uuid.New()
	uploadedBy := uuid.New()
	files := &memFiles{objects: map[string][]byte{"erp-imports/scansource/a.xlsx": []byte("xlsx")}}
	catalog := &memCatalog{supplierID: supplierID, supplierOK: true}
	bus := &memBus{}
	now := time.Date(2026, 11, 13, 15, 31, 20, 0, time.UTC)

	svc := &Service{
		Files:   files,
		Catalog: catalog,
		Events:  bus,
		Read:    func([]byte) (domain.Workbook, error) { return sampleWorkbook(), nil },
		Now:     func() time.Time { return now },
	}

	err := svc.HandleFileReceived(context.Background(), FileReceived{
		JobID:         jobID,
		CorrelationID: correlation,
		SupplierCode:  "scansource",
		BlobContainer: "erp-imports",
		BlobName:      "scansource/a.xlsx",
		UploadedBy:    uploadedBy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bus.completed) != 1 {
		t.Fatalf("completed = %d", len(bus.completed))
	}
	got := bus.completed[0]
	if got.JobID != jobID || got.Stats.OffersUpserted != 1 || got.Stats.ProductsCreatedDraft != 1 {
		t.Fatalf("event = %#v", got)
	}
	if len(catalog.offers) != 1 || catalog.offers[0].PartNumber != "CFQ7TTC0HL8Z:0001:P1Y:Yearly:corporate" {
		t.Fatalf("offers = %#v", catalog.offers)
	}
}

func TestHandleIgnoresOtherSupplier(t *testing.T) {
	bus := &memBus{}
	svc := &Service{Events: bus, Files: &memFiles{}, Catalog: &memCatalog{}, Read: func([]byte) (domain.Workbook, error) {
		t.Fatal("should not read")
		return nil, nil
	}}
	err := svc.HandleFileReceived(context.Background(), FileReceived{SupplierCode: "ingram", JobID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(bus.completed)+len(bus.failed) != 0 {
		t.Fatal("should not publish")
	}
}

func TestHandleParseFailurePublishesFailed(t *testing.T) {
	bus := &memBus{}
	svc := &Service{
		Files:   &memFiles{objects: map[string][]byte{"b/k": []byte("x")}},
		Catalog: &memCatalog{supplierOK: true, supplierID: uuid.New()},
		Events:  bus,
		Read: func([]byte) (domain.Workbook, error) {
			return nil, domain.Permanent(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida")
		},
		Now: func() time.Time { return time.Unix(0, 0).UTC() },
	}
	err := svc.HandleFileReceived(context.Background(), FileReceived{
		SupplierCode:  "scansource",
		BlobContainer: "b",
		BlobName:      "k",
		JobID:         uuid.New(),
		CorrelationID: uuid.New(),
		UploadedBy:    uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(bus.failed) != 1 || bus.failed[0].Stage != "parse" || bus.failed[0].ErrorCode != "unreadable_workbook" {
		t.Fatalf("failed = %#v", bus.failed)
	}
}

func TestHandleMissingSupplier(t *testing.T) {
	bus := &memBus{}
	svc := &Service{
		Files:   &memFiles{objects: map[string][]byte{"b/k": []byte("x")}},
		Catalog: &memCatalog{supplierOK: false},
		Events:  bus,
		Read:    func([]byte) (domain.Workbook, error) { return sampleWorkbook(), nil },
		Now:     func() time.Time { return time.Unix(0, 0).UTC() },
	}
	err := svc.HandleFileReceived(context.Background(), FileReceived{
		SupplierCode: "scansource", BlobContainer: "b", BlobName: "k",
		JobID: uuid.New(), CorrelationID: uuid.New(), UploadedBy: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(bus.failed) != 1 || bus.failed[0].Stage != domain.StageUpsert {
		t.Fatalf("failed = %#v", bus.failed)
	}
}
