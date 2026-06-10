# 05_progress — Fix ambiguous_master_connection (APPEND-ONLY)

## [2026-06-10] Diagnose + fix
- Root cause xác nhận qua code (create_master.go:218-243) + nguồn seed kép (CMS bootstrap + CDS SQL) + thiếu unique constraint.
- Sửa `internal/bootstrap/master_connection.go`: seed-IF-ABSENT + ON CONFLICT DO NOTHING (không ép active mỗi boot, không đẻ master thứ 2).
- Thêm `migrations/schema/cdc_system_model/081_uniq_active_master_connection.sql`: self-heal dedupe (giữ connection thật > marker) + partial unique index.

## [2026-06-10] Verify (exercise-driven, Rule 16-G3/G6)
- `go build ./...` PASS sau sửa master_connection.go.
- BUG CAUGHT khi test: migration bản đầu dùng `status='inactive'` → vi phạm CHECK `connection_registry_status_check` (hợp lệ chỉ active|paused|failed|retired) → sẽ GÃY boot. Đã sửa sang `status='retired'`.
- Re-test trên DB local (tx BEGIN…ROLLBACK): BEFORE 2 active → AFTER self-heal 1 → bootstrap seed-if-absent giữ 1 (không thêm) → INSERT master active thứ 2 bị partial unique index CHẶN. PASS toàn bộ. DB local nguyên trạng (rollback).

## [2026-06-10] Shadow KHÔNG cần fix (verify theo phản hồi user "shadow testing ko lỗi")
- Root cause khác biệt: shadow dùng `resolveShadowConnectionID` (source_object_v2_sync.go:412) = `role_type IN('shadow','mixed') AND status='active' ORDER BY id ASC LIMIT 1` → KHÔNG có guard ambiguous → không bao giờ 409 (chọn im lặng id nhỏ nhất). Master dùng LIMIT 2 + `if len>1 → ErrMasterConnectionAmbiguous`.
- KHÔNG mirror fix: shadow seed có host/port THẬT + DO UPDATE refresh host/port từ cfg.ShadowDB (hợp lệ); đổi sang seed-if-absent/DO NOTHING sẽ đóng băng host/port. Unique-active-shadow index phá multi-shadow + CDS shadow_pg_default.
- KẾT LUẬN: shadow không đổi. Rủi ro tiềm ẩn riêng: resolveShadowConnectionID chọn LIMIT 1 im lặng → nếu >1 shadow active khác target có thể bind nhầm (tùy chọn hardening, không thuộc bug 409 này).

## [2026-06-10] PIVOT: dùng cơ chế shadow cho master (user confirm: master DB chỉ có 1)
- Verify: GetMasterDB(key) bỏ qua key + config MasterDB số ít → single-master; master_connection_id cosmetic → silent-pick an toàn.
- BỎ migration 081 (over-engineered: unique-index hard-code single-master, rủi ro deploy không cần). GIỮ bootstrap seed-IF-ABSENT (hygiene orthogonal).
- Sửa create_master.go resolveMasterConnection: `ORDER BY updated_at DESC,id DESC LIMIT 2` + nhánh ambiguous → `ORDER BY id ASC LIMIT 1` (mirror resolveShadowConnectionID), bỏ ErrMasterConnectionAmbiguous → HẾT 409.
- ErrMasterConnectionAmbiguous + handler branch (master_registry_handler_create.go:80) thành dead nhưng vô hại (vẫn referenced → compile OK) — optional cleanup sau.
