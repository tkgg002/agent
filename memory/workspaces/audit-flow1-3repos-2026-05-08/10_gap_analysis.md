# Gap Analysis

## P0

- Flow 1 FE hiện không build được nên không thể coi là release-ready.
- FE `Step3_Shadow` không khớp payload contract với `RegistryHandler.Register`.
- FE `Step1_Connection` phụ thuộc response field không khớp với `SourcesHandler.Create`.

## P1

- Lịch sử runtime cho thấy introspection route từng bị 404 dù code router hiện đã mount.
- Điều này gợi ý local CMS binary hoặc process lifecycle đang drift so với working tree.

## P2

- Worker test suite không sạch:
  - package `scratch/` làm ô nhiễm `go test ./...`
  - nil logger panic trong `SchemaValidator`
  - regression ở DLQ metadata wrapping

## P3

- Worker runtime có noise vận hành:
  - OTel collector DNS fail
  - NATS publish permission violation
  - transmute numeric casting errors cho master layer
