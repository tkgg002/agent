# Plan — Phase 35 Runtime Schema Follow-through

1. Audit lại các path còn hardcode `public` nhưng vẫn có caller thật sau Phase 34.
2. Sửa `PendingFieldRepo` để hỗ trợ lookup theo schema.
3. Sửa `SchemaInspector` để resolve schema từ metadata V2.
4. Sửa `HandleDiscover` và `HandleBackfill` trong worker sang schema-aware.
5. Verify bằng `gofmt` + `go test` trên cụm package bị chạm.
6. Ghi đủ docs phase và append progress/status/gap.
