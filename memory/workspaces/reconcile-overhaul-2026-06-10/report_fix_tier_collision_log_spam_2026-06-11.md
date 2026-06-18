# report_fix_tier_collision_log_spam_2026-06-11.md — `SELECT COUNT(*) FROM ""` spam

> Muscle:Claude-Opus-4.8 | 2026-06-11 | User report: log cms spam 42601 mỗi poll 5s

## 1. Root cause — LỖI CỦA TÔI (tier collision, nhận trách nhiệm rõ)
Khi thiết kế Segment B (P1) tôi chọn `tier=4` cho `recon_runs` chỉ nhìn producer-side (1/2/3 đã dùng) mà **không grep consumer**: cms `ListBackfillRuns` đã lọc `tier=4` cho backfill-source-ts từ trước → 6 run Segment B (table_name = FQN `schema.table` có dấu chấm) bị BackfillStatus nhặt nhầm → `CountTableRows(FQN)` → `utils.PgIdent` fail-close `""` → `SELECT COUNT(*) FROM ""` (SQLSTATE 42601) bắn vào DB ×6 mỗi lần FE poll 5s.

## 2. Fix 3 tầng (defense in depth)
| Tầng | File | Đổi |
|---|---|---|
| Producer | worker `recon_core.go` | `tierSegmentB` 4 → **5** (grep verified tier 5 free) + comment cảnh báo |
| Consumer (chặn vĩnh viễn) | cms `recon_read_repo_gorm.go` | `ListBackfillRuns` +`WHERE instance_id LIKE 'backfill:%'` — backfill runs luôn có prefix này |
| Caller guard | cms `recon_read_repo_gorm.go` | `CountTableRows` guard `ident == ""` → error sớm, không bắn query lỗi vào DB (+import fmt) |
| Data-fix | recon_runs **25 rows** + report **25 rows** tier 4→5 (phân biệt qua `instance_id NOT LIKE 'backfill:%'`) |

## 3. Verify (bằng chứng thật)
- Build worker + cms PASS; restart worker p4g (PID 90730) + cms p4e (`/health ok`).
- **20s polling thật (FE đang mở): log cms 0 × 42601 — SẠCH**; endpoint backfill-status vẫn 200/16ms.
- Trigger Segment B `b3` → `recon_runs` ghi **tier=5 success** ✅; `SELECT count(*) tier=4 non-backfill` = **0** (không còn rows lẫn).

## 4. Lesson đã ghi
Nhóm 2 Architecture (+1 → 43 pattern): *"Thêm giá trị discriminator vào bảng dùng chung mà không grep consumer lọc theo giá trị đó → collision nhặt nhầm"* — quy tắc: grep mọi `WHERE col = value` trước khi chọn enum value mới; consumer lọc ≥2 điều kiện đặc trưng; fail-close phải guard ở caller.

## 5. Services
Worker p4g 8082 + CMS p4e 8083 + FE 5173 — RUNNING, log sạch.
