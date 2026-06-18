# 09_tasks_solution_recon_v4.md — Thiết kế tổng thể Reconcile V4 (End-to-End)

> Workspace `reconcile-overhaul-2026-06-10` | 2026-06-10 | Muscle:Claude-Opus-4.8
> **Trạng thái: DESIGN — chờ Boss approve. CHƯA code.**
> Đây là MỘT giải pháp duy nhất. Gap đối chuẩn: xem `10_gap_analysis_recon_v4.md`.

---

## 0. Triết lý thiết kế

**Toàn vẹn End-to-End (source ↔ master) = hợp của 2 segment đo được ĐỘC LẬP:**

```
   SOURCE ──Debezium/Kafka──▶ SHADOW ──Transmute──▶ MASTER
      └────── Segment A ───────┘└────── Segment B ────┘
              (ingest path)          (transform path)

   E2E OK  ⇔  A OK ∧ B OK
   E2E lỗi →  định vị NGAY tắc ở đâu: A lỗi = pipeline ingest; B lỗi = transmute
```

Lý do tách segment thay vì so trực tiếp source↔master: (1) so trực tiếp Mongo↔master KHÔNG định vị được điểm tắc (trụ 4 của chuẩn); (2) master là dữ liệu ĐÃ transform (mapping/masking/flatten) — so field trực tiếp với source đòi re-derive toàn bộ transform ở recon job = nhân đôi logic transmute (vi phạm DRY, nguồn bug mới); (3) shadow giữ `_raw_data` + `_hash` + `_gpay_id` làm "điểm tựa trung gian" được thiết kế sẵn cho việc này.

**Self-healing = RE-TRIGGER qua pipeline chuẩn, không bypass.** Dữ liệu sửa lỗi phải đi lại đúng đường (masking/mapping/OCC được áp như mọi event khác). Cụ thể: Segment A dùng **Debezium incremental snapshot signal** (cơ chế chính thống của Debezium cho đúng việc này — handler `cdc.cmd.debezium-signal` ĐÃ CÓ trong worker) thay vì dummy-UPDATE vào source DB — không ghi bất kỳ byte nào vào DB nguồn production (nguyên tắc core-systems, không xâm phạm source); hiệu quả tương đương: connector re-emit full payload cho đúng các key cần sửa.

**Real-time hay batch?** (chốt luôn câu hỏi của blueprint): **hybrid theo mức** — L1 micro-job liên tục (10'), L2 định kỳ (6h, chỉ bảng drift), L3 batch off-peak (02-05h); riêng Segment B thêm chế độ **event-driven**: tự đối soát ngay sau mỗi `cdc.evt.transmute.completed` (close-loop đã có) cho bảng vừa transmute.

---

## 1. Kiến trúc đích

```
                       ┌─────────────────────────────────────────────┐
                       │              ReconCore V4                   │
                       │  (giữ nguyên lock/leader/run-state/report)  │
                       ├──────────────────────┬──────────────────────┤
                       │  Segment A (có sẵn)  │  Segment B (XÂY MỚI) │
                       │  source ↔ shadow     │  shadow ↔ master     │
                       │  Mongo/PG ↔ PG:5436  │  PG:5436 ↔ PG:5434   │
                       ├──────────────────────┴──────────────────────┤
                       │ L1 count window 10' │ L2 chunk-hash 6h      │
                       │ L3 row-diff off-peak│ B: event-driven thêm  │
                       ├─────────────────────────────────────────────┤
                       │   Watermark adaptive (lag-aware)            │
                       ├──────────────┬──────────────┬───────────────┤
                       │ Heal-A:      │ Heal-B:      │ Alert + Lag   │
                       │ dbz signal   │ transmute    │ monitor       │
                       │ re-snapshot  │ SourceIDs    │ (3 điểm đo)   │
                       └──────────────┴──────────────┴───────────────┘
```

Join key đối soát Segment B (đã verify cột thật trên 5436 + 5434):
- Shadow: `_gpay_id BIGINT` (deterministic per source-row), `_source_ts BIGINT`, `_hash`, `_deleted`.
- Master: `_gpay_id` (PK conflict key của transmuter), `_source_ts` (OCC), `_deleted`, `_updated_at`.
→ So theo `(_gpay_id, _source_ts)`: thiếu `_gpay_id` = **missing** (transmute rớt); `_source_ts` master < shadow = **stale** (transmute chưa đuổi kịp/OCC kẹt).

---

## 2. Thiết kế chi tiết + code demo

### 2.1 Segment B — Recon shadow ↔ master (XÂY MỚI, tái dùng engine)

**L1-B (count window, chạy 10' + event-driven sau transmute.completed):**
```sql
-- Phía shadow (5436)                          -- Phía master (5434)
SELECT COUNT(*) FROM "shadow_dev000"."export_jobs"   SELECT COUNT(*) FROM "master_x"."export_jobs"
WHERE _source_ts >= $lo AND _source_ts < $hi          WHERE _source_ts >= $lo AND _source_ts < $hi
  AND _deleted = false                                  AND COALESCE(_deleted,false) = false
```
Lệch ≥ threshold → window drift → đẩy sang L2-B.

**L2-B (chunk hash + drill-down, 6h hoặc khi L1-B drift):**
```sql
-- Cùng 1 câu cả 2 phía — XOR fingerprint theo (_gpay_id, _source_ts):
SELECT COUNT(*) AS cnt,
       COALESCE(bit_xor(hashtextextended(_gpay_id::text || '|' || _source_ts::text, 0)), 0) AS xorh
FROM   {relation}
WHERE  _source_ts >= $lo AND _source_ts < $hi AND COALESCE(_deleted,false) = false;
-- cnt/xorh lệch → drill-down danh sách lệch (1 round-trip, app-side diff 2 set):
SELECT _gpay_id, _source_ts FROM {relation}
WHERE _source_ts >= $lo AND _source_ts < $hi AND COALESCE(_deleted,false)=false
ORDER BY _gpay_id LIMIT 50001;            -- cap 50k+1: tràn cap → hạ window nhỏ hơn, không OOM
```
App-side diff cho ra: `missing_from_master []gpayID`, `stale []gpayID` (ts lệch).
*(Dùng `hashtextextended` built-in PG — không cần extension; 2 phía cùng công thức.)*

**L3-B (row/field-diff thật, off-peak — thay "bucket fingerprint" bằng diff đích danh):**
Chỉ chạy trên các `_gpay_id` lệch từ L2-B. Re-derive expected value từ shadow `_raw_data` qua **chính mapping rules approved** (tái dùng `gjson + ApplyTransform` của transmuter — KHÔNG viết logic transform thứ hai, import cùng package func) rồi so từng cột với master row → ghi `field_diffs JSONB` vào report. Đây là "row-by-row diff" đúng nghĩa của chuẩn mức 3.

**Go skeleton (recon_core.go — thêm, không sửa Tier A):**
```go
// RunSegmentB đối soát shadow↔master cho 1 master binding.
// Tái dùng: withTableLock, beginRun/finishRun, buildWindows, report store.
func (rc *ReconCore) RunSegmentB(ctx context.Context, mb MasterBindingRef) *model.ReconciliationReport {
    lockName := "reconB_" + mb.MasterTable
    ok, unlock := rc.withTableLock(ctx, lockName); if !ok { return nil }
    defer unlock()
    run := rc.beginRun(ctx, mb.MasterTable, tierSegmentB) // tier=4 trong recon_runs
    shadowRel := quoteRelationStr(mb.ShadowSchema, mb.ShadowTable)   // 5436
    masterRel := quoteRelationStr(mb.MasterSchema, mb.MasterTable)   // 5434
    lo, hi := rc.pickScanRangeB(ctx, shadowRel, masterRel)           // watermark §2.2
    for _, w := range rc.buildWindows(lo, hi) {
        sh, _ := rc.pgAgent.CountHashWindow(ctx, rc.shadowDB, shadowRel, w)
        ms, _ := rc.pgAgent.CountHashWindow(ctx, rc.masterDB, masterRel, w)
        if sh.Cnt != ms.Cnt || sh.Xor != ms.Xor {
            missing, stale := rc.diffWindowB(ctx, shadowRel, masterRel, w) // cap 50k
            rc.collect(run, w, missing, stale)
        }
    }
    return rc.finishRunB(ctx, run) // ghi report segment='shadow_master'
}
```
`pgAgent` = `ReconDestAgent` hiện có (đã PG-generic sau fix quoteRelation) — **không cần agent mới**, chỉ thêm method `CountHashWindow` (1 query ở trên).

### 2.2 Watermark adaptive (nâng cấp, áp cả A lẫn B)
```go
// freeze margin theo lag THỰC đo được, kẹp [5m, 60m]:
//   ingestLag    = now − max(_synced_at)  (shadow)
//   transmuteLag = max(_source_ts)shadow − max(_source_ts)master
margin := clamp(5*time.Minute, ingestLag + transmuteLag + 5*time.Minute, 60*time.Minute)
upper  := now.Add(-margin)
```
Hết false-positive khi pipeline lag cao (trụ 1 của chuẩn); 2 con số lag này đồng thời chính là input cho Lag monitoring §2.4.

### 2.3 Self-healing V4 (re-trigger, bỏ heal bypass)

| Phát hiện | Hành động (qua pipeline chuẩn) | Cơ chế đã có sẵn |
|---|---|---|
| A: `missing_from_dest` (source có, shadow thiếu) | Publish `cdc.cmd.debezium-signal` **incremental snapshot** cho table + key-list (chunk ≤1000/lệnh) → connector re-emit → Kafka → sinkworker → shadow → post_ingest transmute | `HandleDebeziumSignal` (worker, đã có) |
| A: orphan (`missing_from_src`) | `UPDATE shadow SET _deleted=true` (soft-delete, KHÔNG hard DELETE) | — (1 UPDATE nhỏ) |
| B: `missing_from_master` / `stale` | Publish `cdc.cmd.transmute` với `SourceIDs=[...gpay_id]` → transmute re-run đúng các row, OCC tự bảo vệ | `TransmuteRequest.SourceIDs` (đã có) |
| Mismatch quá lớn | KHÔNG tự heal: `mismatch > 5000 ID` hoặc `drift_pct > 5%` → chỉ ALERT, chờ operator bấm Heal (chống heal-bão khi sự cố hệ thống) | ngưỡng config |

```go
// Heal-A demo payload (Debezium signal — additional-condition theo PK chunk):
signal := map[string]any{
  "type": "execute-snapshot",
  "data": map[string]any{
    "data-collections": []string{"db1.export_jobs"},
    "type": "incremental",
    "additional-conditions": []map[string]string{{
      "data-collection": "db1.export_jobs",
      "filter": "_id IN ('66f..','66a..', ...)"}}},  // chunk ≤1000
}
nats.Publish("cdc.cmd.debezium-signal", marshal(signal))
// Heal-B demo payload:
nats.Publish("cdc.cmd.transmute", marshal(TransmuteRequest{
  MasterTable: "export_jobs_test", SourceIDs: gpayIDs, TriggeredBy: "recon-heal-b"}))
```
→ Xoá/ngừng đăng ký `ReconHealer` bypass (recon_heal.go giữ file cho DLQ-retry path nếu còn dùng; đường heal chính chuyển hẳn sang re-trigger). Hết phụ thuộc default-Mongo-client (lý do nó chết trên V2), hết bug `extractSourceTsFromDoc` hardcode.

### 2.4 Alert + Lag monitoring (trụ 3a + 4)

**Schema (migration mới, ALTER report + bảng lag):**
```sql
ALTER TABLE cdc_system.cdc_reconciliation_report
  ADD COLUMN IF NOT EXISTS segment TEXT NOT NULL DEFAULT 'source_shadow', -- 'source_shadow'|'shadow_master'
  ADD COLUMN IF NOT EXISTS field_diffs JSONB;                              -- L3-B row-diff output
CREATE TABLE IF NOT EXISTS cdc_system.recon_lag (
  table_name TEXT PRIMARY KEY, ingest_lag_ms BIGINT, transmute_lag_ms BIGINT,
  worker_backlog BIGINT, measured_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
```
- 3 điểm đo mỗi vòng L1: `ingest_lag_ms`, `transmute_lag_ms` (từ §2.2, gần như free), `worker_backlog` (Kafka consumer lag — lấy từ SystemHealth collector đã có).
- **Alert rule** (trong finishRun): `drift_pct > threshold(table, default 0.01%)` HOẶC 2 run liên tiếp `error` HOẶC lag vượt 30' → publish `cdc.evt.alert {table, segment, kind, value}` → `alert_manager` (cms `infra/observability`, đã có) ghi activity + UI badge. Slack webhook = config alert_manager (phase sau, không chặn).

### 2.5 API + FE (mở rộng mặt hiện có, không vẽ trang mới)
- `GET /api/reconciliation/report` thêm field `segment`, `ingest_lag_ms`, `transmute_lag_ms`, `worker_backlog` (JOIN `recon_lag`).
- DataIntegrity.tsx: thêm cột **Segment** (tag `source→shadow` / `shadow→master`) + 2 cột lag; nút Heal hiện theo segment (A→signal, B→transmute); mismatch vượt ngưỡng → badge "cần duyệt heal".
- Dùng lại `job_id` đã trả về để poll `cdc_jobs` (đóng GAP2 cũ): sau trigger check/heal FE poll job tới `done/failed` thay vì setTimeout mù.

---

## 3. Roadmap thực thi (sau khi Boss approve)

| Phase | Nội dung | Ước lượng | Files chính | DoD |
|-------|----------|-----------|-------------|-----|
| **P1** | Segment B engine (L1-B/L2-B + report segment col + wiring masterDB) | 2d | `recon_core.go`(+RunSegmentB), `recon_dest_agent.go`(+CountHashWindow), `worker_server.go`, migration, report API | Tạo lệch giả shadow↔master → L1-B bắt được, report ghi `segment='shadow_master'` đích danh ID |
| **P2** | Self-healing re-trigger A+B + ngưỡng an toàn + gỡ healer bypass | 2d | `recon_handler.go`, heal orchestrator mới (~150 LOC), config | Xoá 5 row master → heal-B tự phục hồi qua transmute; xoá 5 row shadow → heal-A phục hồi qua dbz signal; mismatch giả 10k → KHÔNG tự heal, có alert |
| **P3** | Watermark adaptive + lag monitoring (bảng `recon_lag` + 3 số đo) | 1.5d | `recon_core.go`, migration, SystemHealth glue | Dừng worker 10' → lag tăng, margin giãn, không false-positive; UI thấy lag |
| **P4** | Alert threshold + L3-B row-diff + FE (segment/lag/heal-approve/job-poll) | 1.5d | alert glue, `transmute` reuse cho re-derive, `DataIntegrity.tsx` | Drift > ngưỡng → alert event + badge; L3-B chỉ ra field sai đích danh |
| | **Tổng** | **~7 ngày Muscle** | | Mỗi phase: build+vet+test+E2E evidence+security gate trước khi sang phase sau |

**Điều kiện vận hành đi kèm (Boss quyết):** bật lại `cdc_worker_schedule.reconcile` (hiện disabled); khai DSN cho connection `default_master` nếu muốn recon nguồn PG.

## 4. Những gì KHÔNG làm (và vì sao)
- **Không dummy-UPDATE vào source DB**: ghi vào DB nguồn production để ép CDC = xâm phạm dữ liệu nguồn + tạo event giả trong audit/binlog; Debezium incremental snapshot signal đạt cùng kết quả bằng cơ chế chính thống, hệ đã có handler.
- **Không so field trực tiếp source↔master**: phải nhân đôi logic transform ở recon → 2 nguồn sự thật, bug kép. Re-derive từ shadow `_raw_data` bằng CHÍNH code transmute (import dùng chung) chỉ ở L3-B.
- **Không viết PG source-agent mới cho Segment B**: `ReconDestAgent` sau fix quoteRelation đã PG-generic — dùng cho cả 2 phía.
- **Không đụng luồng source→shadow** (sinkworker/batchBuffer/Debezium config) — pipeline isolation §4.

## Verb chờ Boss
- `approve recon v4` → Muscle thực thi P1 (theo trình tự P1→P4, mỗi phase verify + report riêng).
- `revise <điểm>` → chỉnh thiết kế.
