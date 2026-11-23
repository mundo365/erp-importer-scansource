package domain

import (
	"strings"
)

// Workbook is a map of sheet name → rows (0-based), each row a slice of cell strings.
type Workbook map[string][][]string

func ParseWorkbook(wb Workbook) (ParseResult, error) {
	if wb == nil {
		return ParseResult{}, Permanent(StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida")
	}
	var result ParseResult
	csp, okCSP := findSheet(wb, SheetCSP)
	subs, okSubs := findSheet(wb, SheetSoftSubs)
	perp, okPerp := findSheet(wb, SheetPerpetual)
	if !okCSP && !okSubs && !okPerp {
		return ParseResult{}, Permanent(StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida")
	}
	if okCSP {
		offers, rejected := parseCSP(csp)
		result.Offers = append(result.Offers, offers...)
		result.Stats.RejectedRows = append(result.Stats.RejectedRows, rejected...)
	}
	if okSubs {
		offers, rejected := parseSoftSubs(subs)
		result.Offers = append(result.Offers, offers...)
		result.Stats.RejectedRows = append(result.Stats.RejectedRows, rejected...)
	}
	if okPerp {
		offers, rejected := parsePerpetual(perp)
		result.Offers = append(result.Offers, offers...)
		result.Stats.RejectedRows = append(result.Stats.RejectedRows, rejected...)
	}
	result.Offers = DedupOffers(result.Offers)
	if len(result.Offers) == 0 {
		return result, Permanent(StageParse, "no_valid_offers", "nenhuma oferta valida na planilha")
	}
	return result, nil
}

func findSheet(wb Workbook, name string) ([][]string, bool) {
	if rows, ok := wb[name]; ok {
		return rows, true
	}
	want := strings.ToLower(strings.TrimSpace(name))
	for key, rows := range wb {
		if strings.ToLower(strings.TrimSpace(key)) == want {
			return rows, true
		}
	}
	// Soft Perpétuo may appear without accent
	if name == SheetPerpetual {
		for key, rows := range wb {
			norm := strings.ToLower(strings.TrimSpace(key))
			norm = strings.ReplaceAll(norm, "é", "e")
			norm = strings.ReplaceAll(norm, "ú", "u")
			if norm == "soft perpetuo" {
				return rows, true
			}
		}
	}
	return nil, false
}

func parseCSP(rows [][]string) ([]CanonicalOffer, []RejectedRow) {
	headerIdx, cols := findHeader(rows, 1, []string{"Sku", "Offer Display Name", "Preço Unitário"})
	if headerIdx < 0 {
		return nil, []RejectedRow{{Sheet: SheetCSP, Row: 2, Reason: "missing_header"}}
	}
	var offers []CanonicalOffer
	var rejected []RejectedRow
	for i := headerIdx + 1; i < len(rows); i++ {
		rowNum := i + 1
		row := rows[i]
		sku := cell(row, cols["sku"])
		name := cell(row, cols["offer display name"])
		priceRaw := cell(row, firstCol(cols, "preço unitário", "preco unitario"))
		termHint := cell(row, firstCol(cols, "tempo de contrato"))
		segmentRaw := cell(row, cols["segmento"])
		cycleHint := cell(row, firstCol(cols, "ciclo de pagamento"))
		if strings.TrimSpace(sku) == "" && strings.TrimSpace(priceRaw) == "" {
			continue
		}
		parts, err := ParseCSPSupplierSKU(sku, segmentRaw, termHint, cycleHint)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetCSP, Row: rowNum, Reason: err.Error()})
			continue
		}
		cost, err := ParsePriceBRL(priceRaw)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetCSP, Row: rowNum, Reason: err.Error()})
			continue
		}
		offer, err := NewOfferFromParts(parts, name, cost, sku, SheetCSP, rowNum)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetCSP, Row: rowNum, Reason: err.Error()})
			continue
		}
		offers = append(offers, offer)
	}
	return offers, rejected
}

func parseSoftSubs(rows [][]string) ([]CanonicalOffer, []RejectedRow) {
	headerIdx, cols := findHeader(rows, 1, []string{"Product", "SkuTitle", "Preço"})
	if headerIdx < 0 {
		return nil, []RejectedRow{{Sheet: SheetSoftSubs, Row: 2, Reason: "missing_header"}}
	}
	var offers []CanonicalOffer
	var rejected []RejectedRow
	for i := headerIdx + 1; i < len(rows); i++ {
		rowNum := i + 1
		row := rows[i]
		sku := cell(row, cols["product"])
		name := cell(row, firstCol(cols, "skutitle", "sku title"))
		priceRaw := cell(row, firstCol(cols, "preço", "preco"))
		vigencia := cell(row, firstCol(cols, "vigência", "vigencia"))
		if strings.TrimSpace(sku) == "" && strings.TrimSpace(priceRaw) == "" {
			continue
		}
		parts, err := ParseSoftSubsSKU(sku, vigencia)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetSoftSubs, Row: rowNum, Reason: err.Error()})
			continue
		}
		cost, err := ParsePriceBRL(priceRaw)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetSoftSubs, Row: rowNum, Reason: err.Error()})
			continue
		}
		offer, err := NewOfferFromParts(parts, name, cost, sku, SheetSoftSubs, rowNum)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetSoftSubs, Row: rowNum, Reason: err.Error()})
			continue
		}
		offers = append(offers, offer)
	}
	return offers, rejected
}

func parsePerpetual(rows [][]string) ([]CanonicalOffer, []RejectedRow) {
	headerIdx, cols := findHeader(rows, 0, []string{"SKU", "SkuTitle", "Preço R$"})
	if headerIdx < 0 {
		headerIdx, cols = findHeader(rows, 0, []string{"SKU", "SkuTitle", "Preço"})
	}
	if headerIdx < 0 {
		return nil, []RejectedRow{{Sheet: SheetPerpetual, Row: 1, Reason: "missing_header"}}
	}
	var offers []CanonicalOffer
	var rejected []RejectedRow
	for i := headerIdx + 1; i < len(rows); i++ {
		rowNum := i + 1
		row := rows[i]
		sku := cell(row, firstCol(cols, "sku"))
		name := cell(row, firstCol(cols, "skutitle", "sku title"))
		segmentRaw := cell(row, firstCol(cols, "segment"))
		priceRaw := cell(row, firstCol(cols, "preço r$", "preco r$", "preço", "preco"))
		if strings.TrimSpace(sku) == "" && strings.TrimSpace(priceRaw) == "" {
			continue
		}
		parts, err := ParsePerpetualSKU(sku, segmentRaw)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetPerpetual, Row: rowNum, Reason: err.Error()})
			continue
		}
		cost, err := ParsePriceBRL(priceRaw)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetPerpetual, Row: rowNum, Reason: err.Error()})
			continue
		}
		offer, err := NewOfferFromParts(parts, name, cost, sku, SheetPerpetual, rowNum)
		if err != nil {
			rejected = append(rejected, RejectedRow{Sheet: SheetPerpetual, Row: rowNum, Reason: err.Error()})
			continue
		}
		offers = append(offers, offer)
	}
	return offers, rejected
}

func findHeader(rows [][]string, preferIdx int, required []string) (int, map[string]int) {
	candidates := []int{}
	if preferIdx >= 0 && preferIdx < len(rows) {
		candidates = append(candidates, preferIdx)
	}
	for i := range rows {
		if i == preferIdx {
			continue
		}
		candidates = append(candidates, i)
	}
	for _, idx := range candidates {
		cols := headerMap(rows[idx])
		ok := true
		for _, req := range required {
			if _, found := cols[normalizeHeader(req)]; !found {
				// try without accents / punctuation variants already in normalize
				ok = false
				break
			}
		}
		if ok {
			return idx, cols
		}
	}
	return -1, nil
}

func headerMap(row []string) map[string]int {
	out := make(map[string]int, len(row))
	for i, raw := range row {
		key := normalizeHeader(raw)
		if key == "" {
			continue
		}
		if _, exists := out[key]; !exists {
			out[key] = i
		}
	}
	return out
}

func normalizeHeader(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u",
		"ç", "c",
	)
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func firstCol(cols map[string]int, names ...string) int {
	for _, name := range names {
		if idx, ok := cols[normalizeHeader(name)]; ok {
			return idx
		}
	}
	return -1
}
