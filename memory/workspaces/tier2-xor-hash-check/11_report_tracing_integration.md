# Báo Cáo Thay Đổi: Tích hợp OpenTelemetry Tracing

## 1. Danh sách các file thay đổi & Số dòng code thay đổi

### 1.1. `internal/service/recon/recon_stream.go`
- **Thay đổi:** 
  - Import thêm `"centralized-data-service/pkgs/observability"` và `"go.opentelemetry.io/otel/attribute"`.
  - Hàm `StreamIDsInTimeRange` bổ sung span `mongo.find_batch`.
  - Hàm `streamIDsPostgresInTimeRange` bổ sung span `pg.select_batch`.
- **Số dòng thay đổi:** ~12 dòng.

### 1.2. `internal/service/recon/recon_tier_a.go`
- **Thay đổi:** 
  - Import thêm `"go.opentelemetry.io/otel/attribute"`.
  - Hàm `TimeBoundedDiffMissingFromShadow` bổ sung spans:
    - Parent span: `cdc.recon.time_bounded_diff`.
    - Postgres query span: `pg.query.shadow_ids`.
    - Mongo stream span: `mongo.stream.source_ids`.
- **Số dòng thay đổi:** ~38 dòng.

### 1.3. `internal/handler/recon/recon_heal_v4.go`
- **Thay đổi:**
  - Import thêm `"centralized-data-service/pkgs/observability"`.
  - Nhánh `mode == "full_diff"` của `healSegmentA` bổ sung spans:
    - Scan span: `cdc.recon.heal.full_diff_scan`.
    - Write span: `cdc.recon.heal.direct_write`.
- **Số dòng thay đổi:** ~10 dòng.

---

## 2. Kết Quả Chạy Unit Tests

Tất cả unit tests trong gói `internal/handler/recon` và `internal/service/recon` đều đã compile và pass thành công:

### 2.1. Lệnh 1: `go test -v ./internal/handler/recon/...`
```
=== RUN   TestHealSegmentA_AlwaysFreshScan_LockFail_Noop
--- PASS: TestHealSegmentA_AlwaysFreshScan_LockFail_Noop (0.06s)
=== RUN   TestHealSegmentA_FreshScan_NoReport_NoDrift_Noop
--- PASS: TestHealSegmentA_FreshScan_NoReport_NoDrift_Noop (0.03s)
=== RUN   TestHealSegmentA_RegistryNotFound_Error
--- PASS: TestHealSegmentA_RegistryNotFound_Error (0.03s)
=== RUN   TestHealSegmentA_NatsPublisherNotWired_Error
--- PASS: TestHealSegmentA_NatsPublisherNotWired_Error (0.03s)
=== RUN   TestHealSegmentA_FullDiffMode_InvalidTimeRange
--- PASS: TestHealSegmentA_FullDiffMode_InvalidTimeRange (0.03s)
=== RUN   TestExplodePathToPGPath
--- PASS: TestExplodePathToPGPath (0.00s)
=== RUN   TestValidScanIdent
--- PASS: TestValidScanIdent (0.00s)
=== RUN   TestFlattenJSONWithTypes
--- PASS: TestFlattenJSONWithTypes (0.00s)
=== RUN   TestHandleScanRawData_BackwardCompatibility
--- PASS: TestHandleScanRawData_BackwardCompatibility (0.03s)
=== RUN   TestHandleScanArrayFields_ReplyToAndUnmarshalOrder
--- PASS: TestHandleScanArrayFields_ReplyToAndUnmarshalOrder (0.03s)
PASS
ok  	centralized-data-service/internal/handler/recon	1.057s
```

### 2.2. Lệnh 2: `go test -v ./internal/service/recon/...`
```
=== RUN   TestDestAgent_CountInWindow_Default
--- PASS: TestDestAgent_CountInWindow_Default (0.00s)
...
=== RUN   TestValidatePipelineConnections
--- PASS: TestValidatePipelineConnections (0.00s)
PASS
ok  	centralized-data-service/internal/service/recon	0.620s
```
