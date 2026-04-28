# Plan — Phase 34 Shadow Schema Runtime Hardening

1. Audit các helper/runtime path còn hardcode `public` hoặc query table không qualify schema.
2. Mở thêm lookup `ResolveTargetRoute(targetTable)` trong metadata registry để worker có thể lấy `shadow_schema` từ V2 metadata.
3. Refactor `SchemaValidator` sang introspection theo schema đã resolve.
4. Refactor `CommandHandler` cho các command operator đang chạy thật:
   - `HandleBatchTransform`
   - `HandleScanRawData`
   - `HandlePeriodicScan`
   - `HandleDropGINIndex`
   - `scanFieldsDebezium`
5. Verify bằng `gofmt` + `go test` trên cụm package bị chạm.
6. Append tiến độ/trạng thái/gap vào workspace.
