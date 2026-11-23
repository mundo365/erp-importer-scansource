package domain

import (
	"math/big"
	"strings"
	"unicode"
)

const (
	moneyScale = 10000
	maxMoney   = 999999999999999
)

// ParsePriceBRL parses a ScanSource price cell into scaled int64 (×10⁴).
// Accepts "R$ 1,465.57", "1465.57", "1.465,57" and rounds half-up to 4 decimals.
func ParsePriceBRL(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, "R$", "")
	value = strings.ReplaceAll(value, "r$", "")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, invalid("preço inválido")
	}
	for _, r := range value {
		if unicode.IsSpace(r) {
			continue
		}
		if r == '+' || r == '-' {
			return 0, invalid("preço inválido")
		}
		break
	}
	var b strings.Builder
	for _, r := range value {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	value = b.String()
	if value == "" {
		return 0, invalid("preço inválido")
	}

	normalized, err := normalizeDecimal(value)
	if err != nil {
		return 0, err
	}
	return parseScaledHalfUp(normalized)
}

func normalizeDecimal(value string) (string, error) {
	lastDot := strings.LastIndex(value, ".")
	lastComma := strings.LastIndex(value, ",")
	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot {
			// 1.465,57
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			// 1,465.57
			value = strings.ReplaceAll(value, ",", "")
		}
	case lastComma >= 0:
		frac := value[lastComma+1:]
		if len(frac) > 0 && len(frac) <= 2 && digitsOnly(frac) && strings.Count(value, ",") == 1 {
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	case lastDot >= 0:
		frac := value[lastDot+1:]
		if strings.Count(value, ".") > 1 {
			value = strings.ReplaceAll(value, ".", "")
		} else if len(frac) == 3 && digitsOnly(frac) && !strings.Contains(value[:lastDot], ".") {
			// ambiguous thousands; treat as thousands only when whole has digits and frac len 3 without other separators — keep as integer thousands
			left := value[:lastDot]
			if digitsOnly(left) && len(left) <= 3 {
				// 143.120 could be decimal; keep as decimal
			}
		}
	}
	if strings.HasPrefix(value, ".") {
		value = "0" + value
	}
	whole, frac, _ := strings.Cut(value, ".")
	if whole == "" || !digitsOnly(whole) {
		return "", invalid("preço inválido")
	}
	if frac != "" && !digitsOnly(frac) {
		return "", invalid("preço inválido")
	}
	return whole + "." + frac, nil
}

func parseScaledHalfUp(value string) (int64, error) {
	whole, frac, _ := strings.Cut(value, ".")
	if whole == "" {
		return 0, invalid("preço inválido")
	}
	for len(frac) < 5 {
		frac += "0"
	}
	extra := frac[4]
	frac = frac[:4]
	combined := strings.TrimLeft(whole+frac, "0")
	if combined == "" {
		combined = "0"
	}
	parsed := new(big.Int)
	if _, ok := parsed.SetString(combined, 10); !ok || !parsed.IsInt64() {
		return 0, invalid("preço inválido")
	}
	scaled := parsed.Int64()
	if extra >= '5' {
		scaled++
	}
	if scaled > maxMoney {
		return 0, invalid("preço inválido")
	}
	return scaled, nil
}

func FormatMoney(scaled int64) string {
	if scaled < 0 {
		scaled = 0
	}
	whole := scaled / moneyScale
	frac := scaled % moneyScale
	fracText := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		fracText[i] = byte('0' + frac%10)
		frac /= 10
	}
	return big.NewInt(whole).String() + "." + string(fracText)
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
