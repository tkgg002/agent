# 06 Validation & Verification Plan

## Danh sách kịch bản kiểm thử

| ID | Kịch bản | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|
| TC-01 | Build test backend `centralized-data-service` | `go build ./cmd/...` thành công, không lỗi type/syntax | Exit code 0 | PASS |
| TC-02 | Build test backend `cdc-cms-service` | `go build ./cmd/...` thành công, không lỗi type/syntax | Exit code 0 | PASS |
| TC-03 | Build test frontend `cdc-cms-web` | `tsc -b && vite build` hoàn tất không lỗi TypeScript | Exit code 0 | PASS |
| TC-04 | Unit test `TestHandleBatchTransform_Success` | Mock CTE UPDATE chunked chạy bình thường | PASS | PASS |
| TC-05 | Unit test `TestHandleBatchTransform_ForceMode` | SET chỉ cập nhật field trong `ForceFields`, WHERE là `(TRUE)`, update thành công | PASS | PASS |
| TC-06 | Unit test `TestHandleBatchTransform_UnchunkedFallback` | Chạy unchunked UPDATE an toàn khi không có PK | PASS | PASS |
| TC-07 | Key collision fix trên TableRegistry | Child binding table đọc key `activeTransformJobs[r.id]` độc lập với parent | Code review & build verify | PASS |
| TC-08 | Cast Expression cho timestamptz | Fallback branch sử dụng `::TIMESTAMPTZ` thay vì `::TIMESTAMP` | Code review & build verify | PASS |
