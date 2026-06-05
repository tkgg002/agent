# 10_gap_analysis — Audit Shadow Create Bugs

## GAP-1: Duplicate DDL builder paths
- **Vị trí**: `centralized-data-service/internal/handler/command_handler.go:586-602` (path FE-trigger) **vs** `centralized-data-service/internal/sinkworker/schema_manager.go:231` (path Debezium runtime).
- **Vấn đề**: 2 path build shadow DDL độc lập, không share contract. Bug 2 là hệ quả trực tiếp: sinkworker đúng, handler drift.
- **Nguyên nhân**: Khi thêm V2 anchor (`_gpay_source_id`) và `_source_ts` ở sinkworker, không có rule/test bắt buộc handler đồng bộ.
- **Đề xuất (phase sau)**: Tách interface `ShadowSchemaSpec` (list cột system + index + constraint) đặt trong package shared (`internal/shadow/spec`). Cả 2 path call spec → 1 source of truth. Có thể là phase Q3 refactor.

## GAP-2: `HandleScanFields` cũng dùng `GetActiveRulesBySourceTable`
- **Vị trí**: `command_handler.go:1389`.
- **Vấn đề**: Same root cause logic. Khi user bấm Scan Fields thủ công cho registry id=99 (target=sd_export_jobs_1), nếu query này được gọi → vẫn pull rules của id=42.
- **Đề xuất**: Phase fix mở rộng SOL-1 lên cả line 1389. Cần đọc context của `HandleScanFields` để xác định nó có `SourceObjectID` không.

## GAP-3: Không có integration test cho `HandleCreateDefaultColumns`
- **Vấn đề**: Bug 2 (thiếu `_source_ts`) là regression dễ phát hiện qua test schema diff. Hiện tại chỉ có unit test cho sinkworker upsert.
- **Đề xuất**: Thêm test case `TestHandleCreateDefaultColumns_HasSystemCols` so sánh information_schema.columns sau khi gọi handler ↔ expected set `{_gpay_source_id, _raw_data, _source, _source_ts, _synced_at, _version, _hash, _gpay_deleted, _deleted, _created_at, _updated_at}`.

## GAP-4: Không có lint rule chặn thêm DDL Shadow thiếu cột
- **Vấn đề**: Phòng ngừa regression tương lai.
- **Đề xuất**: Custom static-check (Go AST visitor) scan các string literal có `CREATE TABLE` trong package `handler`/`sinkworker` và assert chứa `_source_ts` + `_gpay_source_id`. Có thể là CI step rẻ.

## GAP-5: Shadow tables hiện hữu (như `sd_export_jobs_1`) đang ở trạng thái mixed schema
- **Vấn đề**: Nếu sinkworker đang ingest vào shadow này, OCC guard tham chiếu `shadow._source_ts` → cột không tồn tại → PG error `column "_source_ts" does not exist` → DLQ.
- **Đề xuất**: MIGR-1 priority cao. Check log DLQ + monitor để xác định scope impact.

## GAP-6: Audit log không trace được "shadow created with N system cols"
- **Vấn đề**: ActivityLog hiện chỉ ghi "register accepted". Không có verify-after-create.
- **Đề xuất**: Sau CREATE TABLE, query information_schema.columns → log số cột system. Catch regression sớm.

## Lesson candidates cho lessons.md (chờ ghi sau khi user duyệt fix)
- **Lesson-001 (Global Pattern [A queries B by NAME-only field instead of ID]) → Result Y [cross-entity bleed when N entities share same NAME]. Đúng**: khi domain cho phép NAME duplicate (vd. source_table có thể được map nhiều lần để tạo nhiều shadow), MUST query theo identity key (id/UUID), không bao giờ chỉ qua NAME.
- **Lesson-002 (Global Pattern [A và B độc lập build cùng 1 schema spec X]) → Result Y [drift inevitable khi spec thay đổi]. Đúng**: extract spec thành single source of truth + test verify A.spec == B.spec.
