package spreadsheet

import (
	"bytes"
	"fmt"
	"strings"

	xls "github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"

	"github.com/mundo365/erp-importer-scansource/internal/domain"
)

var oleMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
var zipMagic = []byte{0x50, 0x4B}

func Read(content []byte) (domain.Workbook, error) {
	if len(content) < 4 {
		return nil, domain.Permanent(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida")
	}

	if isEncryptedOLE(content) {
		return readEncrypted(content)
	}

	switch {
	case isZip(content):
		return readXLSX(content)
	case isOLE(content):
		return readXLS(content)
	default:
		return nil, domain.Permanent(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida")
	}
}

func readXLSX(content []byte) (domain.Workbook, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, domain.PermanentWrap(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida", err)
	}
	defer f.Close()
	wb := domain.Workbook{}
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, domain.PermanentWrap(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida", err)
		}
		wb[name] = rows
	}
	if len(wb) == 0 {
		return nil, domain.Permanent(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida")
	}
	return wb, nil
}

func readXLS(content []byte) (domain.Workbook, error) {
	book, err := xls.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, domain.PermanentWrap(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida", err)
	}
	if book.GetNumberSheets() == 0 {
		return nil, domain.Permanent(domain.StageParse, "encrypted_workbook", "planilha protegida ou ilegivel (OLE sem abas BIFF)")
	}
	wb := domain.Workbook{}
	for i := 0; i < book.GetNumberSheets(); i++ {
		sheet, err := book.GetSheet(i)
		if err != nil {
			return nil, domain.PermanentWrap(domain.StageParse, "unreadable_workbook", "arquivo nao e uma planilha valida", err)
		}
		name := strings.TrimSpace(sheet.GetName())
		if name == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}
		n := sheet.GetNumberRows()
		rows := make([][]string, 0, n)
		for r := 0; r < n; r++ {
			row, err := sheet.GetRow(r)
			if err != nil {
				rows = append(rows, nil)
				continue
			}
			cols := row.GetCols()
			cells := make([]string, len(cols))
			for c, cell := range cols {
				cells[c] = cell.GetString()
			}
			rows = append(rows, cells)
		}
		wb[name] = rows
	}
	return wb, nil
}
