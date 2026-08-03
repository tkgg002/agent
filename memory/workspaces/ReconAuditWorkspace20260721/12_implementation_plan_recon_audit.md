# Kế Hoạch Triển Khai Fix Lỗi Recon Drift & Metric Reporting

Workspace: `ReconAuditWorkspace20260721`

## Proposed Changes

### Centralized Data Service (`internal/service/recon/`)

#### [MODIFY] [recon_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go)
- Sửa hàm `extractMongoIDFromRaw`: Thêm xử lý `Int64OK()`, `Int32OK()`, `DoubleOK()` bằng `strconv.FormatInt` để trích xuất đúng chuỗi số ID (ví dụ `"504"`) thay vì fallback ra `bson.RawValue.String()` (`{"$numberLong":"504"}`).

#### [MODIFY] [recon_stream_bucket_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go)
- Cập nhật struct `ChunkEngineResult` chứa `TotalSrcCount`, `TotalDstCount`, `TotalRecordDiffCount`, `DriftWindowCount`, `MissingCount`, `Drifts`, và `StaleIDs *StaleIDsPayload`.
- Định nghĩa `StaleIDsPayload` với JSON tags `mismatched`, `missing_from_master`, `missing_from_shadow`.
- Khi sub-window bị drift, thực thi drill-down tập hợp các ID bị sai lệch/thiếu.

#### [MODIFY] [recon_job_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon_job_repo.go)
- Cập nhật `ReconJob` struct và `UpdateStatus` lưu `total_record_diff_count`, `source_count`, `dest_count`.

#### [MODIFY] [recon_job_worker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker.go)
- Cập nhật `ReconJobWorker` lưu `source_count`, `dest_count`, `total_record_diff_count`, `stale_ids` vào `recon_jobs` (cột + `result_summary` JSONB) và `cdc_reconciliation_report`.

## Verification Plan

### Automated Tests
- Chạy unit tests: `go test -v ./internal/service/recon/...`
- Đảm bảo 100% unit tests pass.

### Integration Verification
- Build & restart service (`make run`).
- Bắn POST request tới `/api/reconciliation/check?type_recon=hash_window`.
- Đổi soát dữ liệu trong `cdc_system.cdc_reconciliation_report`:
  - `source_count`: 216
  - `dest_count`: 216
  - `diff`: 0
  - `missing_count`: 0
  - OpenTelemetry Traces: `recon.chunk_drift_count = 0`.
