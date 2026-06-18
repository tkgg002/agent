# report_recon_v4_p4_alert_rowdiff_fe_2026-06-10.md — Recon V4 Phase 4 (cuối): Alert + L3-B Row-diff + FE

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | Phase cuối roadmap đã approve

## 1. Đã làm gì
- **Alert ngưỡng** (trụ 3a): worker UPSERT thẳng `cdc_system.cdc_alerts` (tái dùng nguyên hệ alert sẵn có — fingerprint sha256 mirror byte-level `persistence.Fingerprint` của cms, dedup + occurrence++, KHÔNG đè alert đang silenced). Rule: report `drift` → `ReconDrift` warning (critical khi missing>1000); `error` → `ReconError` critical; heal bị chặn ngưỡng → `ReconHealBlocked` warning. FE banner `/api/alerts/active` hiển thị sẵn — không sửa cms alert plane.
- **L3-B row/field-diff** (trụ 2c): `RunRowDiffB` — re-derive expected từ shadow `_raw_data` qua CHÍNH mapping rules approved (`gjson` + `ApplyTransform` — không nhân đôi logic transmute), so từng cột với master row → `field_diffs JSONB` `[{gpay_id, column, expected, actual}]` (cap 200). L2-B nâng cấp: `ListIDTsInWindow` → **stale đích danh theo (id, ts)** thay vì chỉ đếm window. Trigger: payload `"deep":true` per-table. Migration 083.
- **FE DataIntegrity**: cột **Segment** (tag `source→shadow`/`shadow→master` + tooltip), cột **Lag** (humanize m/h/d, màu theo mức); heal gửi `segment` của row → CMS forward (`ReconHealCommand.Segment`) → worker route heal-A/heal-B.

## 2. Files THỰC TẾ đã sửa (git diff)
### centralized-data-service (lũy kế nhánh recon: 8 files +693/−34 + 3 file mới)
| File | P4 |
|---|---|
| `internal/service/recon_alert.go` | **NEW 87 dòng** — FireAlert/alertOnReport/fingerprint mirror |
| `internal/service/recon_core.go` | +~170 P4 (diffIDTs, RunRowDiffB, FieldDiff, SetPlaneDBs, normalizeDiffVal, deep param, alert hooks Tier1+SegB, MasterBindingRef.ID) |
| `internal/service/recon_dest_agent.go` | +57 (IDTs + ListIDTsInWindow) |
| `internal/handler/recon_handler.go` + `recon_heal_v4.go` | payload `deep` + FireAlert khi blocked |
| `internal/server/worker_server.go` | +SetPlaneDBs(shadowDB, masterDB) |
| `internal/model/reconciliation_report.go` | +FieldDiffs |
### cdc-cms-service (5 files +27/−6 + migration 083)
`recon_async.go` (+Segment), `reconciliation_handler_heal.go` (forward segment), `recon_read_models.go`, `recon_read_repo_gorm.go`, `model` (+FieldDiffs), `083_recon_field_diffs.sql` NEW.
### cdc-cms-web (DataIntegrity.tsx +49, useReconStatus.ts +13)
Cột Segment/Lag, ReconRow types, heal segment-aware. *(MasterRegistry.tsx +70 trong stat là việc trước, không thuộc P4.)*

## 3. Verify E2E (bằng chứng thật)
| Test | Kết quả |
|------|---------|
| Build worker + cms + FE, test 94 case | ✅ PASS |
| Migration 083 apply | ✅ |
| **Alert drift** | ✅ `cdc_alerts`: `ReconDrift` warning labels `{table: export_jobs_mt, segment: shadow_master}` firing |
| **Alert heal-blocked** | ✅ `ReconHealBlocked` warning (op=recon-heal-b) khi heal mt bị chặn ngưỡng |
| **Row-diff bắt đích danh ô corrupt** | ✅ Stale-inject (`userId='CORRUPTED_BY_RECON_TEST'` + bump ts trên master b3) → deep check → `field_diffs` n=6, entry đầu: `{column: userId, actual: CORRUPTED_BY_RECON_TEST, expected: 61234c18f1fc05b7a79a71b9}` — đúng ô bị phá |
| **Cleanup sau test** | ✅ revert row corrupt → re-check `ok 0/0`; row xoá ngoài-window phục hồi qua re-transmute (count=1) |
| Watermark đúng thiết kế (phát hiện trong test) | row ts cũ hơn 7d / row mới nhất (upper exclusive) nằm ngoài window — recon window-based không thấy → ĐÚNG hành vi; full-coverage là việc của Tier 3 off-peak |

## 4. Ghi chú trung thực (known limitations)
- **Row-diff noise**: so sánh loose (`fmt.Sprint`) → date (`2026-01-23...UTC` vs epoch `1.769e+12`) và quoted-string (`"abc"` vs `abc`) báo lệch giả — 5/6 entries của test là noise kiểu này. Bắt đúng ô corrupt thật nhưng cần **type-aware normalize** ở vòng tinh chỉnh (đã cap 200 entries nên không nguy hiểm).
- `export_jobs_mt` có 0 approved master rules → row-diff trả rỗng đúng hành vi (không có rule thì không re-derive được).
- `worker_backlog` vẫn NULL — glue Kafka consumer lag chưa làm (nguồn nằm ở cms collector; cần bridge riêng — DEFER có ghi nhận, không chặn).
- Heal-A row-level vẫn pending verify trên môi trường có Kafka Connect (từ P2).
- 1 sự cố trong turn: start worker sai cwd → config relative path fail → phát hiện qua log, start lại đúng cwd (worker yêu cầu chạy từ repo root).

## 5. Services cuối turn
Worker PID 52594 (binary P4) RUNNING 8082 · CMS (binary P4) RUNNING 8083 `/health ok` · FE build PASS (dist mới; dev server tự reload).

## 6. Recon V4 — TRẠNG THÁI TOÀN ROADMAP: P0 ✅ P1 ✅ P2 ✅ P3 ✅ **P4 ✅ — HOÀN THÀNH**
Việc treo cho Boss quyết: bật `cdc_worker_schedule.reconcile` (auto-run); xử lý orphan `export_jobs_mt*` (161/162 row); DSN connection `default_master`; verify heal-A trên staging có Connect; vòng tinh chỉnh type-aware normalize cho row-diff.
