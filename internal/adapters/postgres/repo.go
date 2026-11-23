package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}
	return pool, nil
}

func (r *Repository) FindActiveSupplierID(ctx context.Context, code string) (uuid.UUID, error) {
	var raw string
	err := r.pool.QueryRow(ctx, `
		SELECT id::text FROM suppliers WHERE code = $1 AND active = true
	`, code).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, domain.Permanent(domain.StageUpsert, "supplier_not_found", "fornecedor scansource ativo nao encontrado")
	}
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(raw)
}

func (r *Repository) UpsertOffers(ctx context.Context, supplierID uuid.UUID, offers []domain.CanonicalOffer) (domain.ImportStats, error) {
	stats := domain.ImportStats{}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return stats, err
	}
	defer tx.Rollback(ctx)

	for _, offer := range offers {
		productID, created, err := upsertProduct(ctx, tx, offer)
		if err != nil {
			return stats, domain.PermanentWrap(domain.StageUpsert, "product_upsert_failed", "falha ao gravar produto", err)
		}
		if created {
			stats.ProductsCreatedDraft++
		} else {
			stats.ProductsExisting++
		}
		if err := upsertOffer(ctx, tx, supplierID, productID, offer); err != nil {
			return stats, domain.PermanentWrap(domain.StageUpsert, "offer_upsert_failed", "falha ao gravar oferta", err)
		}
		stats.OffersUpserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ImportStats{}, err
	}
	return stats, nil
}

func upsertProduct(ctx context.Context, tx pgx.Tx, offer domain.CanonicalOffer) (uuid.UUID, bool, error) {
	id := uuid.New()
	var rawID string
	var inserted bool
	err := tx.QueryRow(ctx, `
		INSERT INTO products (
			id, part_number, name, lifecycle_status, billing_mode, installment_count,
			microsoft_product_id, microsoft_sku_id, term, billing_cycle, segment
		) VALUES (
			$1, $2, $3, $4, $5, 0,
			$6, $7, $8, $9, $10
		)
		ON CONFLICT ((lower(part_number))) DO UPDATE SET
			name = EXCLUDED.name,
			microsoft_product_id = EXCLUDED.microsoft_product_id,
			microsoft_sku_id = EXCLUDED.microsoft_sku_id,
			term = EXCLUDED.term,
			billing_cycle = EXCLUDED.billing_cycle,
			segment = EXCLUDED.segment
		RETURNING id::text, (xmax = 0) AS inserted
	`, id, offer.PartNumber, offer.Name, domain.LifecycleDraft, domain.BillingSingle,
		offer.MicrosoftProductID, offer.MicrosoftSkuID, offer.Term, offer.BillingCycle, offer.Segment,
	).Scan(&rawID, &inserted)
	if err != nil {
		return uuid.Nil, false, err
	}
	parsed, err := uuid.Parse(rawID)
	return parsed, inserted, err
}

func upsertOffer(ctx context.Context, tx pgx.Tx, supplierID, productID uuid.UUID, offer domain.CanonicalOffer) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO supplier_offers (id, supplier_id, product_id, cost_price, currency, is_available)
		VALUES ($1, $2, $3, $4::numeric, $5, $6)
		ON CONFLICT (supplier_id, product_id) DO UPDATE SET
			cost_price = EXCLUDED.cost_price,
			currency = EXCLUDED.currency,
			is_available = EXCLUDED.is_available
	`, uuid.New(), supplierID, productID, domain.FormatMoney(offer.CostScaled), domain.CurrencyBRL, offer.IsAvailable)
	return err
}
