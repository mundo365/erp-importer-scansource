package domain

import "testing"

func TestParsePerpetualAlphanumericSku(t *testing.T) {
	skus := []string{
		"DG7GMGF0HPVW-000T-Comm-BR",
		"DG7GMGF0HPVV-0007-Comm-BR",
		"DG7GMGF0HPVV-000W-Char-BR",
	}
	for _, sku := range skus {
		parts, err := ParsePerpetualSKU(sku, "Commercial")
		if err != nil {
			t.Fatalf("%s: %v", sku, err)
		}
		t.Log(parts.PartNumber())
	}
}

func TestParsePerpetualLongNumericSku(t *testing.T) {
	cases := []struct {
		sku string
		id  string
	}{
		{"DG7GMGF0HPVV-00012-Comm-BR", "00012"},
		{"DG7GMGF0HPVV-00011-Char-BR", "00011"},
		{"DG7GMGF0HPVV-00010-Comm-BR", "00010"},
		{"DG7GMGF0HPVV-1-Comm-BR", "0001"},
	}
	for _, tc := range cases {
		parts, err := ParsePerpetualSKU(tc.sku, "Commercial")
		if err != nil {
			t.Fatalf("%s: %v", tc.sku, err)
		}
		if parts.SkuID != tc.id {
			t.Fatalf("%s sku=%s want %s", tc.sku, parts.SkuID, tc.id)
		}
	}
}
