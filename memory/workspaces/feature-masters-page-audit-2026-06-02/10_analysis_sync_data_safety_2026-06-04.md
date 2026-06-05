# 10_analysis_sync_data_safety_2026-06-04.md — Phân tích Data-Safety & Performance: Shadow → Master Sync

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-04
> **Yêu cầu User**: "sync shadow→master thì toàn bộ cơ chế an toàn dữ liệu & performance phải đảm bảo: snapshot, hash, chunk, progress, realtime (oplog), log từng lần sync... phân tích."
> **Phương pháp**: đọc source `transmuter.go`/`transmute_handler.go`/`sinkworker.go`/`transmute_scheduler.go` + verify LIVE trên stack đang chạy (DB read-only + trigger thật). Mọi kết luận có evidence `file:line` hoặc số liệu thực.

---

## 0. Tóm tắt điều hành
Pipeline shadow→master (TransmuterModule) đã có **nền tảng tốt**: chunk pagination, hash change-detection idempotent, soft-delete, realtime per-row, 3 cơ chế trigger. **NHƯNG** có **1 lỗ hổng data-safety nghiêm trọng (false-positive success)** và thiếu **incremental high-water-mark** (perf ở quy mô lớn). Trong phiên này đã **fix 2 bug code thật** (pgx encode + ShadowPK cross-DB) đưa `scanned` từ 0 → 453.

---

## 1. Bug đã phát hiện & xử lý trong phiên (minh hoạ trực tiếp vấn đề data-safety)

| # | Bug | Evidence | Trạng thái |
|---|-----|----------|-----------|
| B1 | `loadRules` pgx encode: `? AS source_object_id` bind int 66 → Postgres suy luận text(OID25) → "cannot find encode plan" → transmute **error** | `transmuter.go:291` | ✅ FIXED `?::bigint` |
| B2 | **ShadowPK detect sai DB**: `loadMaster` probe `information_schema` cột `_gpay_id` trên **systemDB(5433)** nhưng shadow table ở **shadowDb(5436)** → probe luôn fail → fallback `source pk='_id'` (Mongo, NULL trong shadow) → fetch `WHERE (_id)::bigint>0` khớp **0 dòng** → "success scanned=0" | `transmuter.go:220-228` (cũ) | ✅ FIXED hardcode `_gpay_id` (synthetic shadow PK) |
| C1 | **Config**: master `sssss` = `transform_type=flatten` + `transform_spec={}` (rỗng). flatten `BuildEmits` thiếu `explode_path` → **error mỗi dòng** → `processBatch` skip → 453 skipped, **inserted=0**, status=**success** | `flatten.go:66-68`, `transmuter.go:382` | ⚠️ CONFIG (không tự đổi — User dặn "không thay đổi config để đạt kết quả"). sssss muốn sync 1:1 thì để `copy_1_to_1`. |

**Kết quả LIVE sau fix**: `scanned 0 → 453` (đọc đúng shadow). `inserted=0` còn lại 100% do C1 (flatten config), KHÔNG phải code.

---

## 2. Phân tích từng khía cạnh (hiện trạng → verdict → rủi ro → đề xuất)

### 2.1 Snapshot / Initial full load
- **Hiện trạng**: KHÔNG có pha "snapshot" riêng. `Run` luôn khởi tạo `lastGpayID=0` (`transmuter.go:173`) rồi phân trang TOÀN BỘ shadow theo `_gpay_id` mỗi lần chạy (run-now/cron). Realtime (post_ingest) thì chạy theo `_source_ids` cụ thể (incremental).
- **Verdict**: 🟡 Có "full sync" (đóng vai snapshot) nhưng KHÔNG tách pha & KHÔNG resume.
- **Rủi ro**: run-now/cron **quét lại toàn bộ shadow mỗi lần** → ở bảng hàng triệu dòng rất tốn I/O (dù hash chặn ghi thừa). Không có "snapshot xong → chuyển incremental".
- **Đề xuất (P1)**: Lưu high-water-mark `last_gpay_id` vào `sync_runtime_state`; `Run` đọc cursor khởi đầu từ đó (full lần đầu, incremental các lần sau). Tách rõ mode `snapshot` (full) vs `incremental` (từ HWM).

### 2.2 Hash / Change-detection / Idempotency
- **Hiện trạng**: ✅ `computeMasterHash` trên business columns (`transmuter.go:399`); upsert `ON CONFLICT (_source_id) DO UPDATE ... WHERE <t>._hash IS DISTINCT FROM EXCLUDED._hash` (`upsertMaster ~:512`). Dòng không đổi → **không ghi**.
- **Verdict**: ✅ TỐT — idempotent, tránh ghi thừa, an toàn re-run.
- **Rủi ro nhỏ**: nhãn metric lệch — `processBatch:419` coi `RowsAffected==0` (no-op vì hash trùng) là `inserted++`, còn insert/update thật `RowsAffected==1` là `updated++` → số liệu inserted/updated KHÔNG đáng tin.
- **Đề xuất (P2)**: dùng `RETURNING (xmax=0)` để phân biệt insert vs update chuẩn; đếm "unchanged" riêng.

### 2.3 Chunk / Batching / Memory
- **Hiện trạng**: ✅ `batchSize=500` (`transmuter.go:127`); fetch `ORDER BY _gpay_id LIMIT ?` + cursor `> lastGpayID` (`:333-344`); vòng lặp tới khi batch rỗng.
- **Verdict**: ✅ TỐT — bounded memory, keyset pagination (không OFFSET).
- **Đề xuất (P2)**: batchSize nên cấu hình theo binding (bảng rộng → batch nhỏ). Cân nhắc `context` timeout/cancel giữa batch (hiện 5 phút tổng — bảng lớn có thể timeout).

### 2.4 Progress tracking
- **Hiện trạng**: 🟡 Phân tán & không resumable:
  - `sync_runtime_state` (markRuntimeSuccess/Failure/Skipped) — trạng thái cuối.
  - `cdc_activity_log` op=`transmute` (✅ THÊM phiên này) — running→success/failed + stats.
  - `JobMonitor` cập nhật `transmute_schedule.last_status` — **chỉ cron** (run-now không set `schedule_id` → `job_monitor.go:75` skip).
- **Verdict**: 🟡 Có log kết quả nhưng KHÔNG có % tiến trình theo chunk, KHÔNG resume (restart → quét lại từ 0).
- **Đề xuất (P1)**: ghi tiến độ theo chunk (scanned/total, last_gpay_id) vào `sync_runtime_state` để UI hiển thị % và để resume sau crash. run-now cũng nên set `schedule_id` để `last_status` cập nhật.

### 2.5 Realtime (oplog → shadow → master)
- **Hiện trạng**: ✅ `sinkworker.publishTransmuteTrigger` (gate `hasPostIngestSchedule`, cache 30s, fail-open) → `cdc.cmd.transmute-shadow` → `HandleTransmuteShadow` fan-out tới mọi master active+approved → `cdc.cmd.transmute` với `_source_ids` (incremental đúng dòng vừa ghi).
- **Verdict**: ✅ TỐT — realtime per-row, chỉ chạy khi có post_ingest schedule bật (giảm noise). Subscriber đăng ký vô điều kiện (`worker_server.go:412`, không black-hole).
- **Rủi ro**: fail-open → khi gate query lỗi vẫn publish (đúng cho realtime, nhưng có thể publish thừa). Fan-out không dedup nếu nhiều master.
- **Đề xuất (P2)**: thêm metric đếm trigger publish/giây; cân nhắc debounce theo (shadow_table,_source_id) khi burst.

### 2.6 Logging từng lần sync
- **Hiện trạng**: ✅ (sau phiên này) mỗi lần `HandleTransmute` ghi 1 row `cdc_activity_log` (op=transmute, target=master, triggered_by=run-now-actor/scheduler/sinkworker-hook, status running→success/failed, rows_affected, details: scanned/inserted/updated/skipped/rule_misses/type_errors/duration). + `transmute complete` log dòng.
- **Verdict**: ✅ ĐỦ để audit từng lần. (Trước phiên: ❌ KHÔNG ghi gì → bug âm thầm — đã vá.)

### 2.7 ⚠️ DATA-SAFETY: False-positive success (NGHIÊM TRỌNG NHẤT — đúng nỗi lo của User)
- **Hiện trạng**: transmute báo `status=success` ngay cả khi **0 dòng tới master** vì lỗi/skip:
  - B2 (đã fix): scanned=0 vẫn success.
  - C1 (live): **453 scanned, 453 skipped, 0 inserted, status=success** — flatten thiếu explode_path, `BuildEmits` error MỖI dòng nhưng `processBatch:382` chỉ `skipped++ continue` (không fatal) → run vẫn "success".
  - Tổng quát: type-errors / non-nullable-missing / transform-error → skip dòng, run vẫn success.
- **Verdict**: ❌ Lỗ hổng lớn — operator/UI thấy "xanh" nhưng dữ liệu KHÔNG sang. Trùng anti-pattern lessons `#false-positive-success`.
- **Đề xuất (P0)**:
  1. Nếu `scanned>0 && (inserted+updated)==0 && skipped==scanned` → đánh dấu `status=failed` (hoặc `degraded`) + `error_message` nêu lý do chủ đạo (vd "all rows skipped: flatten explode_path missing").
  2. Nếu transform `BuildEmits` trả error cho **mọi** dòng (cùng 1 lỗi config) → fail nhanh cả run (không skip im lặng 453 lần).
  3. Activity log đính kèm `skip_reasons` (top-N) để chẩn đoán 1 dòng.

### 2.8 Validation cấu hình tại thời điểm tạo Master
- **Hiện trạng**: ⚠️ flatten `ValidateSpec` đòi `explode_path` (`flatten.go:49-57`), nhưng `sssss` vẫn tạo được với spec `{}` → validation KHÔNG được enforce lúc tạo/approve master.
- **Đề xuất (P0/P1)**: CMS khi create/approve master phải gọi `strategy.ValidateSpec(transform_spec)` theo `transform_type`; reject nếu flatten thiếu explode_path. Chặn config sai từ gốc.

### 2.9 Concurrency / Fencing / Transactionality / Deletes
- **Fencing**: ✅ scheduler `FOR UPDATE SKIP LOCKED` + fencing token (`transmute_scheduler.go`). run-now KHÔNG lock → run-now + cron đồng thời có thể double-process (an toàn nhờ hash-idempotent, nhưng tốn công). **Đề xuất (P2)**: advisory lock theo master khi transmute.
- **Transactionality**: ⚠️ upsert **per-row** (`processBatch:413`), không bọc transaction theo batch → batch fail giữa chừng để lại dữ liệu một phần (re-run idempotent sửa được, nhưng không atomic). **Đề xuất (P2)**: cân nhắc upsert theo batch trong 1 tx.
- **Deletes**: ✅ `_deleted` mang sang master (soft-delete, `:410`).
- **Ordering**: ✅ keyset theo `_gpay_id` tăng dần.

---

## 3. Bảng ưu tiên

| Ưu tiên | Hạng mục | Lý do |
|---------|----------|-------|
| **P0** | 2.7 Chặn false-positive success (scanned>0 & inserted+updated=0 → failed/degraded + reason) | An toàn dữ liệu — operator phải biết sync rỗng |
| **P0** | 2.8 Enforce `ValidateSpec` khi tạo/approve master (flatten cần explode_path) | Chặn config sai từ gốc |
| **P1** | 2.1+2.4 High-water-mark incremental + progress resumable | Performance quy mô lớn + UX tiến trình |
| **P1** | run-now set `schedule_id` để JobMonitor cập nhật last_status | Quan sát đầy đủ |
| **P2** | 2.2 metric insert/update chuẩn; 2.9 batch-tx + advisory lock; 2.3 batch config; 2.5 debounce realtime | Tinh chỉnh độ chính xác & hiệu năng |

---

## 4. Đã verify LIVE phiên này
- Stack chạy: CMS:8083, NATS:14222, PG 5433/5434/5436, worker `go run cmd/worker` (đã restart với code fix).
- Trigger thật `cdc.cmd.transmute` master=sssss → activity_log: `scanned 0→453` sau fix B1+B2; `inserted=0` do C1 (flatten config).
- 2 fix code (B1,B2) đã build/test PASS; chưa commit (chờ User — xem report).
