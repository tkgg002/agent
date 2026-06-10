# 09_tasks_solution — Fix ambiguous_master_connection

## Thay đổi (2 file source)
| File | Thay đổi |
|---|---|
| `internal/bootstrap/master_connection.go` | `ON CONFLICT DO UPDATE...active` → **seed-IF-ABSENT** (`SELECT ... WHERE NOT EXISTS(active master) ON CONFLICT DO NOTHING`). Không ép active mỗi boot, không tạo master thứ 2, không clobber operator. |
| `migrations/schema/cdc_system_model/081_uniq_active_master_connection.sql` (mới) | (1) **self-heal**: giữ 1 master active (ưu tiên connection thật > marker `default_master`, rồi updated_at/id), số còn lại → `status='retired'`. (2) **partial unique index** `uniq_active_master_connection` enforce ≤1 active master. |

## Vì sao chặn được tái diễn
- Index ⇒ DB **không cho phép** >1 row `role_type='master' AND status='active'` → `create_master.go:240 (len(rows)>1)` không bao giờ đúng nữa → hết 409.
- Seed-if-absent ⇒ CMS không còn đẻ/đè master thứ 2 ⇒ không xung đột với index.
- Self-heal trong migration ⇒ **tự dọn dup có sẵn trên testing/prod khi deploy** (không cần ops chạy SQL tay).

## Deploy
- **Auto**: service boot kế tiếp → migrate runner áp `081_*` qua advisory lock → self-heal + tạo index. Idempotent (chạy lại = no-op).
- Migration chạy trong tx; `CREATE UNIQUE INDEX` (non-concurrent) trên bảng nhỏ → tức thì.

## Rollback
- Index: `DROP INDEX IF EXISTS cdc_system.uniq_active_master_connection;`
- Code: revert 2 file. (Các row đã `retired` giữ nguyên — đúng ý, không cần khôi phục.)

## Verify (đã chạy, Rule 16-G3/G6)
- `go build ./...` PASS. Test tx trên DB local: 2 active → 1 (self-heal) → bootstrap không thêm → INSERT thứ 2 bị index chặn. Bắt + sửa bug `status='inactive'` (CHECK chỉ cho active|paused|failed|retired).

## Security (Rule 8 self-check)
- SQL tĩnh, không có input người dùng trong migration/bootstrap → không SQLi. `secret_ref` chỉ là metadata (không mở connection). Không tăng attack surface.

## Shadow — KHÔNG mirror fix (đã verify 2026-06-10)
- Shadow KHÔNG bị 409: `resolveShadowConnectionID` (source_object_v2_sync.go:412) dùng `LIMIT 1 ORDER BY id ASC`, KHÔNG có guard ambiguous (khác master LIMIT 2 + error). → miễn nhiễm cấu trúc.
- KHÔNG áp seed-if-absent/unique-index cho shadow: shadow seed có host/port thật + DO UPDATE refresh host/port từ `cfg.ShadowDB` (hợp lệ) — đổi sẽ đóng băng host/port; unique-active-shadow phá multi-shadow + CDS `shadow_pg_default`.
- Tùy chọn (không thuộc bug này): hardening chống "chọn nhầm shadow im lặng" bằng deterministic preference, KHÔNG đụng refresh host/port.

---
## CẬP NHẬT [2026-06-10] — PHƯƠNG ÁN CUỐI (pivot, supersede phần index ở trên)
User xác nhận **master DB chỉ có 1**. GetMasterDB(key) bỏ qua key → master_connection_id là metadata cosmetic. → Dùng **cơ chế shadow** cho master (đơn giản, Rule 6), bỏ index/migration.

**Net change (2 file, uncommitted):**
- `internal/app/commands/create_master.go` — `resolveMasterConnection`: `LIMIT 2 + if len>1 ambiguous` → `ORDER BY id ASC LIMIT 1` (mirror `resolveShadowConnectionID`). HẾT 409.
- `internal/bootstrap/master_connection.go` — giữ seed-IF-ABSENT (hygiene; tùy chọn revert về DO UPDATE nếu muốn tối giản tuyệt đối).
- **ĐÃ XOÁ** migration `081_uniq_active_master_connection.sql` (over-engineered).

**Verify**: go build PASS; DB test 2 master active → resolver trả 1 row no-error.
**Dead-but-harmless**: `ErrMasterConnectionAmbiguous` + nhánh 409 ở `master_registry_handler_create.go:80` không còn được resolver trả về (vẫn compile vì còn reference) — optional cleanup.
**Testing dupes**: còn 2 row master nhưng vô hại (resolver tolerant). Muốn registry gọn: `UPDATE cdc_system.connection_registry SET status='retired' WHERE role_type='master' AND status='active' AND id NOT IN (SELECT min(id) FROM cdc_system.connection_registry WHERE role_type='master' AND status='active');` (tùy chọn, không bắt buộc).
