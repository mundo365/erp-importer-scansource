package domain

import "testing"

func TestParseCSPSupplierSKU(t *testing.T) {
	parts, err := ParseCSPSupplierSKU("CFQ7TTC0HL8Z-1-P1Y-Annual-BR", "Commercial", "Anual", "Anual")
	if err != nil {
		t.Fatal(err)
	}
	if got := parts.PartNumber(); got != "CFQ7TTC0HL8Z:0001:P1Y:Yearly:corporate" {
		t.Fatalf("part number = %s", got)
	}
}

func TestParseCSPTrienni(t *testing.T) {
	parts, err := ParseCSPSupplierSKU("CFQ7TTC0RN26-1-P3Y-Trienni-BR", "Commercial", "Trienal", "Trienal")
	if err != nil {
		t.Fatal(err)
	}
	if parts.Term != TermP3Y || parts.Cycle != BillingCycleTriennial {
		t.Fatalf("term/cycle = %s/%s", parts.Term, parts.Cycle)
	}
	if parts.SkuID != "0001" {
		t.Fatalf("sku = %s", parts.SkuID)
	}
}

func TestParseCSPAlphanumericSku(t *testing.T) {
	parts, err := ParseCSPSupplierSKU("CFQ7TTC0LFLX-009T-P3Y-Trienni-BR", "Commercial", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if parts.SkuID != "009T" {
		t.Fatalf("sku = %s", parts.SkuID)
	}
}

func TestParseSoftSubsSKU(t *testing.T) {
	parts, err := ParseSoftSubsSKU("DG7GMGF0GJC2-0003-Comm-BR", "Annual")
	if err != nil {
		t.Fatal(err)
	}
	if got := parts.PartNumber(); got != "DG7GMGF0GJC2:0003:P1Y:Yearly:corporate" {
		t.Fatalf("part number = %s", got)
	}

	parts, err = ParseSoftSubsSKU("DG7GMGF0GJC2-0001-Comm-BR", "Triennial")
	if err != nil {
		t.Fatal(err)
	}
	if got := parts.PartNumber(); got != "DG7GMGF0GJC2:0001:P3Y:Triennial:corporate" {
		t.Fatalf("part number = %s", got)
	}
}

func TestParsePerpetualSKU(t *testing.T) {
	parts, err := ParsePerpetualSKU("DG7GMGF0PN5J-0002-Comm-BR", "Commercial")
	if err != nil {
		t.Fatal(err)
	}
	if got := parts.PartNumber(); got != "DG7GMGF0PN5J:0002:one-time:one-time:corporate" {
		t.Fatalf("part number = %s", got)
	}

	parts, err = ParsePerpetualSKU("DG7GMGF0PN5J-0002-Char-BR", "")
	if err != nil {
		t.Fatal(err)
	}
	if parts.Segment != SegmentCharity {
		t.Fatalf("segment = %s", parts.Segment)
	}

	parts, err = ParsePerpetualSKU("DG7GMGF0PN5J-0002-Educ-BR", "Education")
	if err != nil {
		t.Fatal(err)
	}
	if parts.Segment != SegmentAcademic {
		t.Fatalf("segment = %s", parts.Segment)
	}
}

func TestMapSegmentGovernment(t *testing.T) {
	seg, err := MapSegment("Public Sector")
	if err != nil || seg != SegmentGovernment {
		t.Fatalf("seg=%s err=%v", seg, err)
	}
}
