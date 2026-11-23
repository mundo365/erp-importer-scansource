# erp-importer-scansource

Worker Go (Linux/container) que consome `import.file.received.v1`, importa a planilha ScanSource (abas CSP, Soft Subs e Soft Perpétuo) no Postgres do `erp-core` e publica `import.completed.v1` / `import.failed.v1`.

## Pré-requisitos

Subir o Compose do `erp-core` (Postgres, MinIO/`pgsty/silo`, RabbitMQ). Reutiliza as mesmas variáveis locais.

```bash
cp .env.example .env
go run ./cmd/importer
```

Health: `GET http://localhost:8081/healthz`.

A imagem **não** inclui Excel nem LibreOffice. O upload no `erp-core` só aceita `.xlsx` ZIP ou `.xls` BIFF **sem DRM**.

## Fluxo

1. `erp-core` valida o arquivo (rejeita DRM com 422), grava no S3 e publica `import.file.received.v1`.
2. Este worker ignora `supplier_code` diferente de `scansource`.
3. Baixa o objeto (`blob_container`/`blob_name`), parseia a planilha, faz upsert de `products` + `supplier_offers`.
4. Publica `import.completed.v1` com `stats`, ou `import.failed.v1` com `stage` `parse`|`upsert`.

Não regenera `price_lists`. Produtos Microsoft novos entram como `draft`.

## Planilha ScanSource com DRM

A lista oficial costuma vir com IRM/DRM. Na SPA (`erp-web`), baixe `Unlock-ScanSource.ps1`, rode no PowerShell com Excel 365 e envie o `*-sem-protecao.xlsx`. Se um OLE criptografado chegar à fila, o job falha com `encrypted_workbook`.
