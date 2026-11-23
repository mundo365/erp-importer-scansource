package spreadsheet

import (
	"bytes"

	"github.com/richardlehane/mscfb"
)

func isZip(content []byte) bool {
	return bytes.HasPrefix(content, zipMagic)
}

func isOLE(content []byte) bool {
	return bytes.HasPrefix(content, oleMagic)
}

func isEncryptedOLE(content []byte) bool {
	if !isOLE(content) {
		return false
	}
	doc, err := mscfb.New(bytes.NewReader(content))
	if err != nil {
		return false
	}
	hasEncryptedPackage := false
	hasWorkbook := false
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		switch entry.Name {
		case "EncryptedPackage", "DRMEncryptedDataSpace", "DRMEncryptedTransform":
			hasEncryptedPackage = true
		case "Workbook", "Book":
			hasWorkbook = true
		}
	}
	return hasEncryptedPackage && !hasWorkbook
}
