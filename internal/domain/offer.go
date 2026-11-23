package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type RejectedRow struct {
	Sheet  string `json:"sheet,omitempty"`
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type CanonicalOffer struct {
	PartNumber         string
	Name               string
	MicrosoftProductID string
	MicrosoftSkuID     string
	Term               string
	BillingCycle       string
	Segment            string
	CostScaled         int64
	IsAvailable        bool
	SourceSKU          string
	Sheet              string
	Row                int
}

type ParseStats struct {
	RejectedRows []RejectedRow
}

type ParseResult struct {
	Offers []CanonicalOffer
	Stats  ParseStats
}

type ImportStats struct {
	OffersUpserted       int `json:"offers_upserted"`
	ProductsCreatedDraft int `json:"products_created_draft"`
	ProductsExisting     int `json:"products_existing"`
	RejectedRows         int `json:"rejected_rows"`
}

type ProductUpsert struct {
	ID                 uuid.UUID
	PartNumber         string
	Name               string
	MicrosoftProductID string
	MicrosoftSkuID     string
	Term               string
	BillingCycle       string
	Segment            string
	Created            bool
}

type OfferUpsert struct {
	ID          uuid.UUID
	SupplierID  uuid.UUID
	ProductID   uuid.UUID
	CostScaled  int64
	Currency    string
	IsAvailable bool
}

func NewOfferFromParts(parts CanonicalParts, name string, costScaled int64, sourceSKU, sheet string, row int) (CanonicalOffer, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return CanonicalOffer{}, invalid("nome inválido")
	}
	if costScaled < 0 {
		return CanonicalOffer{}, invalid("preço inválido")
	}
	return CanonicalOffer{
		PartNumber:         parts.PartNumber(),
		Name:               name,
		MicrosoftProductID: parts.ProductID,
		MicrosoftSkuID:     parts.SkuID,
		Term:               parts.Term,
		BillingCycle:       parts.Cycle,
		Segment:            parts.Segment,
		CostScaled:         costScaled,
		IsAvailable:        true,
		SourceSKU:          sourceSKU,
		Sheet:              sheet,
		Row:                row,
	}, nil
}

func DedupOffers(offers []CanonicalOffer) []CanonicalOffer {
	order := make([]string, 0, len(offers))
	byPN := make(map[string]CanonicalOffer, len(offers))
	for _, offer := range offers {
		key := strings.ToLower(offer.PartNumber)
		if _, ok := byPN[key]; !ok {
			order = append(order, key)
		}
		byPN[key] = offer
	}
	out := make([]CanonicalOffer, 0, len(order))
	for _, key := range order {
		out = append(out, byPN[key])
	}
	return out
}
