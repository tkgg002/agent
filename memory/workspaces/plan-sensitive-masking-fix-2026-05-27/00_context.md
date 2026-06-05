# 00_context — Sensitive Masking Compliance Fix

## Trigger
User report 2026-05-27: chức năng Sensitive Masking đang ghi đè chuỗi cứng `"***"` vào DB đích (shadow PG) trong pipeline CDC. Cần rà soát + nâng cấp để tuân thủ pháp lý mới của Việt Nam.

## Bối cảnh pháp lý
- **Luật số 91/2025/QH15** — Luật Bảo vệ dữ liệu cá nhân (BVDLCN).
  - Điều 13: Quyền chỉnh sửa + tính chính xác (Accuracy) của dữ liệu cá nhân.
  - Tổ chức xử lý dữ liệu phải đảm bảo dữ liệu chính xác và có thể audit.
- **Nghị định số 356/2025/NĐ-CP** — Hướng dẫn thi hành Luật BVDLCN, yêu cầu biện pháp kỹ thuật bảo vệ (mã hóa, khử định danh, kiểm soát truy cập).
- **Văn bản hợp nhất 25/VBHN-NHNN** — An toàn bảo mật dịch vụ trực tuyến ngành Ngân hàng:
  - Phân vùng mạng + kiểm soát truy cập.
  - Nghiêm cấm lưu trữ trái phép dữ liệu nhận biết KH ở phân vùng nguy cơ cao.
- **Chế tài**: phạt hành chính lên đến **3 tỷ VND** hoặc **5% doanh thu năm liền kề**.

## Vấn đề hiện trạng (evidence từ Explore subagent)

### M1 — Hardcode `"***"` literal
- `centralized-data-service/internal/service/masking_service.go:91` `MaskFieldSample()` return `"***"`.
- `masking_service.go:133` `maskMapRecursive()` set `out[key] = "***"`.
- `masking_service.go:152-153` `maskAnyRecursive()` return `"***"`.
- `masking_service.go:71,77` `MaskJSONPayload()` dùng `"***"` cho invalid JSON.

### M2 — Không có `mask_strategy` per-field
- `cdc-cms-service/migrations/schema/core/001_init_schema.sql:67` chỉ có `sensitive_fields JSONB DEFAULT '[]'` — 1 strategy duy nhất (replace `***`).
- `cdc_mapping_rules` (line 112-149) **không có** column `mask_strategy`, `mask_format`, `hash_salt_ref`.

### M3 — Apply masking ở write-path (DB persistence)
- `centralized-data-service/internal/service/dynamic_mapper.go:67,114` mask `rawData` TRƯỚC khi marshal vào `_raw_data` column.
- → Dữ liệu gốc bị PHÁ HỦY tại shadow, không thể audit Accuracy theo Điều 13 Luật 91/2025.

### M4 — Không có API CRUD masking config
- `cdc-cms-service/internal/` không có endpoint cho `sensitive_fields` hoặc `mask_strategy`.
- Update qua SQL migration thủ công → vi phạm separation of concerns.

### M5 — Test coverage rời rạc
- Không có `masking_service_test.go` riêng.
- Test trong `batch_buffer_test.go`, `recon_handler_test.go`, `text_sanitizer_test.go` chỉ assert `"***"` literal → confirm anti-pattern.

### M6 — Không có design doc
- Không có thư mục `docs/` cho masking. Không có ADR hiện hữu.

## Service scope
- **centralized-data-service**: Core CDC worker — sửa `masking_service.go`, `dynamic_mapper.go`, `batch_buffer.go`, `recon_heal.go`, `kafka_consumer.go`.
- **cdc-cms-service**: Admin backend — thêm migration + API CRUD masking config.
- **cdc-cms-web**: Admin UI — thêm trang config masking strategy per-field.

## Constraints (theo Note của User)
- ✓ Đọc lesson global trước (đã làm; có L-2026-05-26-metric-defined-but-never-set + lesson 63 về silent skip → reminder không repeat anti-pattern).
- ✓ Tuân thủ §1+§12 (Brain plan-only, không touch source code .go/.ts/.sql).
- ✓ Code demo chi tiết trong markdown block.
- ✓ Verify command định lượng cho mỗi gap.
- ✓ Service work verify (build + vet + test) trước khi báo done (Muscle phase).
- ✓ Có file `report_sensitive_masking_fix_2026-05-27.md`.
- ✓ Không cheat DB/config (không workaround, fix root cause: thay đổi schema + masking strategy đúng chuẩn).

## Evidence base reference
- Audit Explore subagent thorough scan 3 service (centralized-data-service + cdc-cms-service + cdc-cms-web).
- Reference plan trước: `plan-cdc-qa-gap-fix-2026-05-27` (composite score progression).
- Lesson global: `agent/memory/global/lessons.md`.
