# report_dbshadow_cdcinternal_fix_2026-06-04.md

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-04
> Phản hồi 4 vấn đề User. Báo cáo dựa trên kết quả THỰC (PREPARE/build/vet/restart).

## Đã thực thi & VERIFY (turn này)

### #3a — db→shadow `POST /api/mapping-rules` vỡ 42601 (CRITICAL, đã unblock)
- **Root cause**: `internal/app/commands/create_mapping_rule.go` — INSERT liệt kê **18 cột** nhưng VALUES chỉ `15 ? + 2 NOW() = 17` biểu thức (thiếu 1 `?`) → `INSERT has more target columns than expressions (42601)`.
- **Quan trọng**: file này **KHÔNG nằm trong thay đổi của tôi** (git chỉ thấy create_master.go + create_schedule.go bị sửa) → **bug COMMITTED sẵn**, không phải do phiên này. Anh test shadow mapping giờ mới chạm.
- **Fix**: thêm 1 `?` → `16 ? + 2 NOW() = 18` = 18 cột.
- **Verify**: `PREPARE` câu INSERT mới → OK (hết 42601); `PREPARE` bản cũ → tái hiện đúng `42601`. CMS `go build`=0, restart CMS (health=200) → db→shadow create unblocked.

### #3b — `cdc_internal` hồi sinh ("_test phải tỉnh táo")
- **Root cause**: `test/internal/app/commands/approve_schema_proposal_integration_test.go` chạy `CREATE SCHEMA IF NOT EXISTS cdc_internal` + tạo `cdc_internal.shadow_users` — `cdc_internal` đã bị `DROP` ở migration 038 (deprecated, mọi thứ chuyển sang `cdc_system`). Test này khi chạy trên DB chung sẽ **tái tạo schema deprecated** (drift bug). (Runtime DB hiện KHÔNG còn cdc_internal — đã xác nhận.)
- **Fix**: đổi `cdc_internal` → `shadow_e2e` (schema test hợp lệ); GIỮ comment lịch sử về migration 037. `go vet` test PASS.
- Còn lại (TODO nhỏ): `centralized-data-service/scripts/smoke_failover.sh` default `cdc_internal.shadow_test_users` (overridable, không phải runtime).

## Files đã thay đổi (turn này)
| File | Repo | Thay đổi | ~LoC |
|------|------|----------|------|
| `internal/app/commands/create_mapping_rule.go` | cdc-cms-service | #3a: thêm 1 `?` vào VALUES | +1/-1 |
| `test/internal/app/commands/approve_schema_proposal_integration_test.go` | cdc-cms-service | #3b: cdc_internal→shadow_e2e (3 chỗ data + comment cảnh báo) | +3/-3 |

## CHƯA thực thi — đã PLAN cụ thể (xem `02_plan_mapping_master_redesign_2026-06-04.md`)
### #1 — Redesign `mapping_rule_master` (link mapping_v2_id, JOIN, filter shadow_binding_id, không copy)
### #2 — Bỏ "Scan Array (Flatten)" khỏi Master (master không đụng shadow DB)
- **Lý do CHƯA execute ngay**: #1 đụng lại **2 query worker** (`transmuter.loadRules` + `master_ddl_generator`) vừa ổn định qua nhiều vòng debug (encode + ShadowPK). Sai 1 chỗ → vỡ transmute lại. Theo nguyên tắc "không làm vỡ + verify từng bước", cần execute cẩn thận (build+live-verify transmute sau mỗi thay đổi worker), KHÔNG gấp cuối phiên.
- Plan đã viết tới từng dòng (migration 075 + clone INSERT...SELECT + GET JOIN + worker JOIN + domain/repo/handler/FE + gỡ flatten route/handler/shadowDB). Sẵn sàng execute.

## Trạng thái service
- CMS: restart, health=200 (chạy #3a). Worker: UP (chạy code P0 data-safety phiên trước).
- Toàn bộ thay đổi (gồm #3a/#3b + phần trước) vẫn **uncommitted** (chưa commit theo §8, chờ User).
