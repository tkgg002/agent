# Validation

## Repo verification

- `cdc-cms-service`
  - `go test ./...` → PASS.
- `cdc-cms-web`
  - `npm run build` → FAIL.
  - TypeScript errors tập trung ở:
    - `src/pages/flow1/Flow1Layout.tsx`
    - `src/pages/flow1/Step1_Connection.tsx`
    - `src/pages/flow1/Step3_Shadow.tsx`
- `centralized-data-service`
  - `go build ./cmd/worker ./cmd/admin-api ./cmd/sinkworker ./cmd/profile_table` → PASS.
  - `go test ./...` → FAIL.
  - `go test ./internal/handler` → FAIL tại `TestExtractDLQMetadata_NonJSONValue`.
  - `go test ./internal/service` → FAIL:
    - `TestConnectionManager_*` bị sandbox chặn connect localhost:5433.
    - `TestSchemaValidatorDriftDetection` panic nil logger trong `schema_validator.go`.
  - `go test ./...` còn fail thêm vì package `scratch/` chứa nhiều file `main` trùng nhau.

## Runtime / UI verification

- FE dev server đang listen ở `localhost:5173`.
- CMS binary đang listen ở `*:8083` theo `lsof`.
- Browser check:
  - mở `http://localhost:5173/flow1/step-1` bị redirect về `/login`.
  - login page render được; wizard không verify end-to-end nếu không có token hợp lệ.
- CMS logs cho thấy:
  - có nhiều request `/api/v1/wizard/sessions/*`, `/api/v1/system/connectors`, `/api/v1/source-objects/*`.
  - có lịch sử `GET /api/v1/introspection/mongo/databases` trả `404`.
- Worker logs cho thấy transmute runtime vẫn chạy, nhưng có lỗi dữ liệu thật:
  - nhiều `ERROR: invalid input syntax for type numeric: "8999/100"` tại `internal/service/transmuter.go:464`.
  - OTel upload lỗi DNS `otel-collector`.
  - có `nats: permissions violation` trong startup log.
