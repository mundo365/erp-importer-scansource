package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var microsoftIDPattern = regexp.MustCompile(`^[A-Z0-9]+$`)

type CanonicalParts struct {
	ProductID string
	SkuID     string
	Term      string
	Cycle     string
	Segment   string
}

func (p CanonicalParts) PartNumber() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", p.ProductID, p.SkuID, p.Term, p.Cycle, p.Segment)
}

func NormalizeMicrosoftSkuID(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" || !microsoftIDPattern.MatchString(value) {
		return "", invalid("sku Microsoft inválido")
	}
	// Short numeric ids pad to 4 (1 → 0001). Longer ScanSource ids like 00012 stay as-is.
	if isDigits(value) && len(value) < 4 {
		return strings.Repeat("0", 4-len(value)) + value, nil
	}
	return value, nil
}

func NormalizeProductID(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" || !microsoftIDPattern.MatchString(value) {
		return "", invalid("produto Microsoft inválido")
	}
	return value, nil
}

func MapSegment(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "_", " ")
	switch {
	case value == "" || value == "br":
		return "", invalid("segmento inválido")
	case strings.HasPrefix(value, "comm") || strings.Contains(value, "commercial") || value == "corporate":
		return SegmentCorporate, nil
	case strings.HasPrefix(value, "educ") || strings.Contains(value, "education") || strings.Contains(value, "academic"):
		return SegmentAcademic, nil
	case strings.HasPrefix(value, "char") || strings.Contains(value, "charity"):
		return SegmentCharity, nil
	case strings.HasPrefix(value, "gov") || strings.Contains(value, "government") || strings.Contains(value, "public"):
		return SegmentGovernment, nil
	default:
		return "", invalid("segmento inválido")
	}
}

func MapBillingCycle(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", invalid("ciclo inválido")
	}
	lower := strings.ToLower(value)
	switch {
	case lower == "monthly" || lower == "mensal":
		return BillingCycleMonthly, nil
	case lower == "yearly" || lower == "annual" || lower == "anual":
		return BillingCycleYearly, nil
	case strings.HasPrefix(lower, "trienni") || lower == "triennial" || lower == "trienal":
		return BillingCycleTriennial, nil
	case lower == "none" || lower == "nenhum":
		return BillingCycleNone, nil
	case lower == "one-time" || lower == "onetime" || lower == "única" || lower == "unica":
		return BillingCycleOneTime, nil
	default:
		return "", invalid("ciclo inválido")
	}
}

func MapTerm(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", invalid("prazo inválido")
	}
	upper := strings.ToUpper(value)
	lower := strings.ToLower(value)
	switch {
	case upper == TermP1M || lower == "mensal" || lower == "monthly":
		return TermP1M, nil
	case upper == TermP1Y || lower == "anual" || lower == "annual" || lower == "yearly":
		return TermP1Y, nil
	case upper == TermP3Y || strings.HasPrefix(lower, "trien"):
		return TermP3Y, nil
	case lower == "one-time" || lower == "onetime" || lower == TermOneTime:
		return TermOneTime, nil
	default:
		return "", invalid("prazo inválido")
	}
}

// ParseCSPSupplierSKU parses CFQ7TTC0HL8Z-1-P1Y-Annual-BR style SKUs.
// Segment comes from the sheet column, not the SKU.
func ParseCSPSupplierSKU(raw, segmentRaw, termHint, cycleHint string) (CanonicalParts, error) {
	parts, err := splitSupplierSKU(raw)
	if err != nil {
		return CanonicalParts{}, err
	}
	productID, err := NormalizeProductID(parts[0])
	if err != nil {
		return CanonicalParts{}, err
	}
	skuID, term, cycle, err := extractCSPMiddle(parts[1:])
	if err != nil {
		return CanonicalParts{}, err
	}
	if term == "" {
		term, err = MapTerm(termHint)
		if err != nil {
			return CanonicalParts{}, err
		}
	}
	if cycle == "" {
		cycle, err = MapBillingCycle(cycleHint)
		if err != nil {
			return CanonicalParts{}, err
		}
	}
	segment, err := MapSegment(segmentRaw)
	if err != nil {
		return CanonicalParts{}, err
	}
	return CanonicalParts{
		ProductID: productID,
		SkuID:     skuID,
		Term:      term,
		Cycle:     cycle,
		Segment:   segment,
	}, nil
}

// ParseSoftSubsSKU parses DG7GMGF0GJC2-0003-Comm-BR and derives term/cycle from vigência.
func ParseSoftSubsSKU(raw, vigencia string) (CanonicalParts, error) {
	parts, err := splitSupplierSKU(raw)
	if err != nil {
		return CanonicalParts{}, err
	}
	if len(parts) < 3 {
		return CanonicalParts{}, invalid("sku ScanSource inválido")
	}
	productID, err := NormalizeProductID(parts[0])
	if err != nil {
		return CanonicalParts{}, err
	}
	skuID, err := NormalizeMicrosoftSkuID(parts[1])
	if err != nil {
		return CanonicalParts{}, err
	}
	segment, err := MapSegment(parts[2])
	if err != nil {
		return CanonicalParts{}, err
	}
	term, cycle, err := termCycleFromVigencia(vigencia)
	if err != nil {
		return CanonicalParts{}, err
	}
	return CanonicalParts{
		ProductID: productID,
		SkuID:     skuID,
		Term:      term,
		Cycle:     cycle,
		Segment:   segment,
	}, nil
}

// ParsePerpetualSKU parses Soft Perpétuo SKUs with one-time term/cycle.
func ParsePerpetualSKU(raw, segmentRaw string) (CanonicalParts, error) {
	parts, err := splitSupplierSKU(raw)
	if err != nil {
		return CanonicalParts{}, err
	}
	if len(parts) < 2 {
		return CanonicalParts{}, invalid("sku ScanSource inválido")
	}
	productID, err := NormalizeProductID(parts[0])
	if err != nil {
		return CanonicalParts{}, err
	}
	skuID, err := NormalizeMicrosoftSkuID(parts[1])
	if err != nil {
		return CanonicalParts{}, err
	}
	segment := strings.TrimSpace(segmentRaw)
	if segment == "" && len(parts) >= 3 {
		segment = parts[2]
	}
	mapped, err := MapSegment(segment)
	if err != nil {
		return CanonicalParts{}, err
	}
	return CanonicalParts{
		ProductID: productID,
		SkuID:     skuID,
		Term:      TermOneTime,
		Cycle:     BillingCycleOneTime,
		Segment:   mapped,
	}, nil
}

func termCycleFromVigencia(raw string) (term, cycle string, err error) {
	cycle, err = MapBillingCycle(raw)
	if err != nil {
		return "", "", err
	}
	switch cycle {
	case BillingCycleYearly:
		return TermP1Y, cycle, nil
	case BillingCycleTriennial:
		return TermP3Y, cycle, nil
	case BillingCycleMonthly:
		return TermP1M, cycle, nil
	default:
		return "", "", invalid("vigência inválida")
	}
}

func splitSupplierSKU(raw string) ([]string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, invalid("sku ScanSource inválido")
	}
	parts := strings.Split(value, "-")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) < 2 {
		return nil, invalid("sku ScanSource inválido")
	}
	last := cleaned[len(cleaned)-1]
	if len(last) == 2 && isAlpha(last) {
		cleaned = cleaned[:len(cleaned)-1]
	}
	if len(cleaned) < 2 {
		return nil, invalid("sku ScanSource inválido")
	}
	return cleaned, nil
}

func extractCSPMiddle(parts []string) (skuID, term, cycle string, err error) {
	if len(parts) == 0 {
		return "", "", "", invalid("sku ScanSource inválido")
	}
	termIdx, cycleIdx := -1, -1
	for i, part := range parts {
		if termIdx < 0 {
			if mapped, mapErr := MapTerm(part); mapErr == nil && (strings.EqualFold(part, TermP1M) || strings.EqualFold(part, TermP1Y) || strings.EqualFold(part, TermP3Y) || strings.EqualFold(part, TermOneTime)) {
				term = mapped
				termIdx = i
				continue
			}
		}
		if cycleIdx < 0 {
			if mapped, mapErr := MapBillingCycle(part); mapErr == nil {
				lower := strings.ToLower(part)
				if lower == "monthly" || lower == "annual" || lower == "yearly" || lower == "none" ||
					strings.HasPrefix(lower, "trienni") || lower == "triennial" || lower == "mensal" || lower == "anual" || lower == "trienal" ||
					lower == "one-time" || lower == "onetime" {
					cycle = mapped
					cycleIdx = i
				}
			}
		}
	}
	skuParts := make([]string, 0, 1)
	for i, part := range parts {
		if i == termIdx || i == cycleIdx {
			continue
		}
		skuParts = append(skuParts, part)
	}
	if len(skuParts) == 0 {
		return "", "", "", invalid("sku Microsoft inválido")
	}
	skuID, err = NormalizeMicrosoftSkuID(skuParts[0])
	if err != nil {
		return "", "", "", err
	}
	return skuID, term, cycle, nil
}

func isDigits(value string) bool {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
}

func isAlpha(value string) bool {
	for _, r := range value {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return value != ""
}
