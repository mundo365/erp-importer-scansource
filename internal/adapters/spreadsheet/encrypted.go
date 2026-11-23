package spreadsheet

import "github.com/mundo365/erp-importer-scansource/internal/domain"

// readEncrypted is a safety net: erp-core should reject DRM at upload.
// Linux containers cannot open Microsoft IRM/DRM workbooks.
func readEncrypted([]byte) (domain.Workbook, error) {
	return nil, domain.Permanent(
		domain.StageParse,
		"encrypted_workbook",
		"planilha protegida por DRM; no Excel 365 salve uma copia como Pasta de Trabalho do Excel (.xlsx) sem protecao e reenvie",
	)
}
