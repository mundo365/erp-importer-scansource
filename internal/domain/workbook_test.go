package domain

import "testing"

func TestParseWorkbookSheets(t *testing.T) {
	wb := Workbook{
		"CSP": {
			{},
			{"Sku", "Offer Display Name", "Preço Unitário", "Tempo de contrato", "Segmento", "Categoria", "Ciclo de pagamento", "Billing Plan"},
			{"CFQ7TTC0HL8Z-1-P1Y-Annual-BR", "10-Year Audit Log Retention Add On", "R$ 143.12", "Anual", "Commercial", "Audit", "Anual", ""},
			{"BAD", "Sem parse", "10.00", "Anual", "Commercial", "x", "Anual", ""},
		},
		"Soft Subs": {
			{},
			{"Product", "SkuTitle", "Preço", "Vigência"},
			{"DG7GMGF0GJC2-0003-Comm-BR", "Azure SQL Edge - 1 year", "R$ 814.33", "Annual"},
			{"", "", "", ""},
		},
		"Soft Perpétuo": {
			{"SKU", "SkuTitle", "Segment", "Preço R$", "Categoria"},
			{"DG7GMGF0PN5J-0002-Comm-BR", "Access LTSC 2024", "Commercial", "R$ 1,546.85", "Office"},
		},
		"Cotação": {{"ignored"}},
	}

	result, err := ParseWorkbook(wb)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Offers) != 3 {
		t.Fatalf("offers = %d %#v", len(result.Offers), result.Offers)
	}
	if len(result.Stats.RejectedRows) != 1 {
		t.Fatalf("rejected = %#v", result.Stats.RejectedRows)
	}
	if result.Offers[0].PartNumber != "CFQ7TTC0HL8Z:0001:P1Y:Yearly:corporate" {
		t.Fatalf("csp pn = %s", result.Offers[0].PartNumber)
	}
}

func TestParseWorkbookZeroCostNone(t *testing.T) {
	wb := Workbook{
		"CSP": {
			{},
			{"Sku", "Offer Display Name", "Preço Unitário", "Tempo de contrato", "Segmento", "Categoria", "Ciclo de pagamento", "Billing Plan"},
			{"CFQ7TTC0AAAA-1-P1Y-None-BR", "Free addon", "0", "Anual", "Commercial", "x", "None", ""},
		},
	}
	result, err := ParseWorkbook(wb)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Offers) != 1 || result.Offers[0].CostScaled != 0 || !result.Offers[0].IsAvailable {
		t.Fatalf("offer = %#v", result.Offers)
	}
	if result.Offers[0].BillingCycle != BillingCycleNone {
		t.Fatalf("cycle = %s", result.Offers[0].BillingCycle)
	}
}

func TestParseWorkbookUnreadable(t *testing.T) {
	_, err := ParseWorkbook(Workbook{"Cotação": {{"x"}}})
	if err == nil {
		t.Fatal("expected error")
	}
	permanent, ok := err.(*PermanentError)
	if !ok || permanent.Code != "unreadable_workbook" {
		t.Fatalf("err = %v", err)
	}
}
