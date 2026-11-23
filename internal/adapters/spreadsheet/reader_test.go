package spreadsheet

import (
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

func TestReadXLSX(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	name := "CSP"
	idx, err := f.NewSheet(name)
	if err != nil {
		t.Fatal(err)
	}
	f.SetActiveSheet(idx)
	_ = f.SetSheetRow(name, "A1", &[]any{"", "", ""})
	_ = f.SetSheetRow(name, "A2", &[]any{"Sku", "Offer Display Name", "Preço Unitário", "Tempo de contrato", "Segmento", "Categoria", "Ciclo de pagamento", "Billing Plan"})
	_ = f.SetSheetRow(name, "A3", &[]any{"CFQ7TTC0HL8Z-1-P1Y-Annual-BR", "Audit", "143.12", "Anual", "Commercial", "x", "Anual", ""})
	_ = f.DeleteSheet("Sheet1")

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	wb, err := Read(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.ParseWorkbook(wb)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Offers) != 1 {
		t.Fatalf("offers = %d", len(result.Offers))
	}
}

func TestReadGarbage(t *testing.T) {
	_, err := Read([]byte("not-an-excel"))
	if err == nil {
		t.Fatal("expected error")
	}
}
