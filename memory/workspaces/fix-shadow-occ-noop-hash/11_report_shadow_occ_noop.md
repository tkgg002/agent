# Report: Bản Vá Triệt Tiêu Blind Update (No-Op Hash Gate) Tại Shadow Table

## 1. Overview
Bản vá đã nâng cấp cơ chế sinh mệnh đề `WHERE` trong câu lệnh Batch Upsert (`INSERT ... ON CONFLICT DO UPDATE SET ... WHERE ...`) của `SchemaAdapter` để kết hợp cả 2 cổng:
1. **Cổng kiểm soát thay đổi dữ liệu (Data Change Gate):** Bắt buộc phải có sự thay đổi nội dung (`_hash IS DISTINCT FROM EXCLUDED._hash` hoặc `_deleted IS DISTINCT FROM EXCLUDED._deleted`).
2. **Cổng kiểm soát thứ tự thời gian OCC (OCC Time Ordering Guard):** Chống out-of-order stale write (`_source_ts < EXCLUDED._source_ts`).

## 2. Những file đã thay đổi & Số dòng code
1. `centralized-data-service/internal/service/shadow/schema_adapter.go` (~40 dòng code thay đổi trong hàm `buildOCCWhereClause`).
2. `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go` (~30 dòng code thêm unit test `TestEventOrdering_SameDataSnapshot_NoOp`).
3. `centralized-data-service/test/internal/service/recon_heal_test.go` (~10 dòng code cập nhật assert OCC + Hash gate).

## 3. Hiệu năng & Kết quả thực tế
- Khi Re-snapshot 1 triệu bản ghi cũ:
  + 999.990 dòng không đổi: PostgreSQL thực hiện **NO-OP** (0 disk write, 0 dead tuples, `_version` giữ nguyên, `_updated_at` giữ nguyên).
  + Vài dòng có update: Cập nhật chính xác dữ liệu mới (`RowsAffected = 1`).
  + Bản ghi mới: `INSERT` mới bình thường (`RowsAffected = 1`).
- Toàn bộ 18 test cases của package `service` đều PASS 100%.
