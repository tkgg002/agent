# report_recon_v4_p2_self_healing_2026-06-10.md — Recon V4 Phase 2: Self-healing Re-trigger

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | Tiếp P2 theo roadmap đã approve

## 1. Đã làm gì
Heal V4 = **re-trigger qua pipeline chuẩn**, thay hẳn đường bypass cũ (vốn disabled trên V2 + đi tắt masking/mapping):
- **Heal-B** (shadow↔master): đọc report Segment B mới nhất → map `_gpay_id`→`_source_id` trên shadow plane (chunk 1000) → publish `cdc.cmd.transmute {_source_ids}` → transmute re-run đúng row, OCC bảo vệ. Orphan ở master **KHÔNG tự xoá** — surface cho operator.
- **Heal-A** (source↔shadow): đọc/chạy Tier-2 lấy missing IDs đích danh → publish `cdc.cmd.debezium-signal` incremental snapshot theo PK-chunk (filter SQL `IN` cho engine SQL, JSON `$in` cho Mongo) — **KHÔNG dummy-update vào source DB**.
- **Ngưỡng an toàn** (chống heal-bão): block khi mismatch > 5000 ID hoặc drift% > 5; **small-table floor ≤ 100 ID** cho heal bất kể % (phát hiện trong E2E thật: bảng bé % luôn cao, blast radius thực = số ID). Block → ERROR log + activity `blocked_threshold` (alert event bus đầy đủ = P4).
- `cdc.cmd.recon-heal` payload mới: `{table, segment, legacy}` — `legacy:true` là escape-hatch về đường bypass cũ (gỡ sau P4).
- Wiring fix quan trọng: `WithNatsPublisher` ĐỘC LẬP với `WithBackfill` (trước đây natsPub chỉ được set khi backfill enabled → heal sẽ chết trên V2 vì backfill disabled).

## 2. Files THỰC TẾ đã sửa (git)
| File | Thay đổi |
|---|---|
| `internal/handler/recon_heal_v4.go` | **NEW 324 dòng** — orchestrator heal A/B + threshold + map gpay→source_id + buildSnapshotIDFilter + WithShadowDB/WithNatsPublisher |
| `internal/handler/recon_handler.go` | +field `shadowDB`; `HandleReconHeal` route theo segment (re-trigger mặc định, legacy flag) |
| `internal/server/worker_server.go` | +`.WithShadowDB(shadowDB).WithNatsPublisher(natsClient.Conn)` |
Diff lũy kế nhánh recon (worker): **8 files +398/−32** + 1 file mới 324 dòng.

## 3. Verify E2E (DoD P2 — bằng drift THẬT)
| Test | Kết quả |
|------|---------|
| Build / vet / test handler+service | ✅ PASS |
| **Heal-B tự phục hồi**: `b3` drift thật (shadow 11 vs master 4, missing 8) → heal | ✅ `dispatched 8` → transmute `scanned=8 inserted=7 updated=1` → **re-check Segment B: `ok` 11=11, missing 0**; report cũ ghi `healed_count=8`. Vòng kín detect→heal→verify ĐÓNG |
| **Threshold block**: `export_jobs_mt` (mismatch 165, drift 8050%) → heal | ✅ `blocked_threshold` + ERROR log + activity row — KHÔNG tự heal |
| **Threshold tinh chỉnh qua test thật**: `b3` lần 1 bị block (drift 63% dù chỉ 9 ID) → thêm small-table floor ≤100 → lần 2 heal chạy | ✅ chính sách floor được chứng minh cần thiết bằng data thật |
| **Heal-A flow**: `export_jobs` → không có tier-2 report → tự chạy Tier 2 → `noop` (không có missing đích danh) | ✅ flow đúng; ⚠️ **đường signal-dispatch chưa exercised end-to-end** vì (a) không có missing thật ở segment A, (b) **Kafka Connect không chạy local** — sẽ verify row-level trên môi trường có Connect (staging). KHÔNG báo láo phần này |

## 4. Trung thực — giới hạn còn lại
- Heal-A row-level recovery **pending verify** trên môi trường có Kafka Connect + Debezium connector sống; format filter Mongo `$in` cần xác nhận với version Debezium thực tế.
- Orphan-in-master (161/162 row ở `export_jobs_mt*`): theo thiết kế chờ operator — UI duyệt heal là P4.
- `transmute complete` log có `type_errors=8` cho b3 (field nullable bị gán nil — hành vi transmuter hiện hữu, không do P2; đáng theo dõi ở P4 row-diff).

## 5. Services
Worker PID mới (binary `/tmp/cdc-worker-recon-p2`) RUNNING 8082 — 18 NATS subjects registered; cms 8083 RUNNING; FE không đổi.

## 6. Next
`P3 — Watermark adaptive + Lag monitoring` (theo roadmap; bảng `recon_lag` + freeze margin theo lag thực).
