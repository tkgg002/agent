# report_recon_v4_p3_watermark_lag_2026-06-10.md — Recon V4 Phase 3: Watermark adaptive + Lag monitoring

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | Tiếp P3 theo roadmap đã approve

## 1. Đã làm gì
- **Watermark adaptive** (trụ 1 chuẩn — moving target): `adaptiveFreeze(lagMs) = clamp(5m, 5m+lag, 60m)` áp cho CẢ 2 segment — `pickScanRange` (A) dùng ingest-lag, `RunSegmentB` dùng transmute-lag. Lag cao → margin tự giãn → không false-positive khi pipeline đang đuổi.
- **Lag monitoring** (trụ 4): bảng mới `cdc_system.recon_lag` (migration 082) — 2 số đo ghi mỗi vòng recon, gần như free vì tận dụng max-ts queries sẵn có:
  - `ingest_lag_ms` = max(_source_ts)source − max(_source_ts)shadow (Segment A đo)
  - `transmute_lag_ms` = max(_source_ts)shadow − max(_source_ts)master (Segment B đo)
  - `worker_backlog` nullable — glue Kafka consumer lag ở P4.
- **API đọc**: cms `LatestReport` (1) JOIN `recon_lag` trả 3 field lag; (2) **fix latest-per-segment**: `DISTINCT ON (target_table, segment)` — row Segment B mới không che mất row Segment A cùng bảng.
- `lagBetween` chống nhiễu: vế zero (bảng rỗng/không đo được) hoặc clock-skew âm → 0, không phạt margin.

## 2. Files THỰC TẾ đã sửa (git)
### centralized-data-service
| File | Thay đổi |
|---|---|
| `internal/service/recon_core.go` | +~63 dòng P3 (adaptiveFreeze, lagBetween, upsertReconLag whitelist-col, pickScanRange + RunSegmentB dùng adaptive + ghi lag) — lũy kế file +330 |
| `internal/service/recon_lag_test.go` | **NEW 50 dòng** — unit test adaptiveFreeze (4 case clamp) + lagBetween (5 case gồm negative-path) |
Lũy kế nhánh recon worker: 8 files +459/−34 + 2 file mới (heal_v4 324, lag_test 50).
### cdc-cms-service
| File | Thay đổi |
|---|---|
| `migrations/schema/recon_dlq/082_recon_lag.sql` | **NEW** — bảng recon_lag |
| `internal/infra/persistence/recon_read_repo_gorm.go` | +8/−2 — JOIN recon_lag + DISTINCT ON (table, segment) |
| `internal/app/queries/recon_read_models.go` | +4 — 3 lag fields |
| `internal/model/reconciliation_report.go` | +2 — field Segment (scan r.*) |

## 3. Verify (bằng chứng thật)
| Bước | Kết quả |
|------|---------|
| Build worker + cms | ✅ PASS cả 2 |
| Unit test `TestAdaptiveFreeze` + `TestLagBetween` | ✅ PASS (9 case, gồm clamp 60m + negative/zero guard) |
| Migration 082 apply 5433 | ✅ CREATE TABLE |
| Worker restart (binary P3) + trigger A+B | ✅ `recon_lag` 7 bảng có SỐ ĐO THẬT: `export_jobs` ingest=0 (đuổi kịp); `export_jobs_mt` transmute_lag=413,918,534ms (~4.8 ngày — đúng tình trạng lệch nặng đã biết); `b3`=62,460,794ms (~17.3h); bảng khoẻ=0 |
| Adaptive hành vi | lag 4.8d → freeze clamp 60m (unit test chứng minh công thức; số lag thật nuôi đúng input) |
| Query per-segment + JOIN lag (đúng SQL API chạy) | ✅ trả 12 rows: cùng bảng có cả row `source_shadow` lẫn `shadow_master` riêng biệt + cột lag đính kèm |
| CMS restart binary mới + `/health` | ✅ `{"service":"cdc-cms","status":"ok"}` (PID 47749) |

## 4. Ghi chú trung thực
- `worker_backlog` chưa có nguồn (Kafka consumer lag glue = P4) — cột NULL, đã khai trong API.
- "UI thấy lag": API đã trả field; FE render cột = P4 (đúng phân phase design).
- `b3` lag 17.3h dù count khớp 11=11: max `_source_ts` shadow mới hơn master — row mới nhất shadow chưa được transmute lại (OCC giữ bản cũ hơn ở master cho row đó) → đáng soi ở P4 row-diff; lag metric làm đúng việc: chỉ ra điểm cần nhìn.

## 5. Services
Worker PID 47320 (binary P3) RUNNING 8082; CMS PID 47749 (binary P3) RUNNING 8083 — cả 2 build từ code mới; FE 5173 không đổi.

## 6. Next
`P4 — Alert event bus + L3-B row-diff + FE` (cột segment/lag, heal theo segment, duyệt heal vượt ngưỡng, job-poll, worker_backlog glue).
