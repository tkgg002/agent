# 03_implementation — Chức năng HEAL Shadow→Master (manual-trigger, robust, auditable)

> Quyết định user: **viết heal, kích hoạt mới chạy, KHÔNG auto-heal** (auto rủi ro + khó audit/debug/checklog).

## Hiện trạng (đã có, không dựng trùng)
- **CMS**: `POST /api/reconciliation/heal/:table` body `{segment:"shadow_master", reason}` → `ReconHealCommand` → worker.
- **Worker** `recon_heal_v4.go healSegmentB`: đọc report Segment B mới nhất → lấy `missing_ids` (_gpay_id) → map sang `_source_id` → publish `cdc.cmd.transmute {_source_ids}` theo chunk (OCC bảo vệ) → mark `healed_at/healed_count`.
- **FE**: nút "Chữa lành" (DataIntegrity) gọi endpoint trên. ⇒ **đã manual-trigger** (đúng yêu cầu).

## Điểm yếu cần "viết lại cho robust"
- **F1 phụ thuộc report**: không có report Segment B → từ chối ("chạy recon-check trước"). Report stale → heal sai/thiếu.
- **F2 over-report**: window-recon (`check_type=segment_b_window`) báo `missing_count=170` (cả bảng) thay vì 7 thật → vẫn chạy nhưng kém minh bạch; hoặc nếu report không liệt kê ids → `noop` dù thực có gap.
- **F3 audit defer**: chỉ log "dispatched N"; healed-count thật phải chờ recon B kế tiếp → khó đối soát ngay.

## Giải pháp (1 hướng — robust + auditable, vẫn MANUAL)
Sửa `healSegmentB` thành **self-contained, không phụ thuộc report**:
1. **Tính gap TƯƠI tại thời điểm heal** (không đọc report cũ): worker đã có `shadowDB` (đọc shadow `_source_id`) + handle master/dest (đọc master `_id`). Diff trực tiếp `shadow._source_id ∉ master._id` → ra **đúng N record thiếu**. (Nếu thiếu handle dest → fallback: publish full re-transmute `cdc.cmd.transmute {master_table}` không `_source_ids`, OCC tự lọc — vẫn guaranteed close.)
2. **Re-transmute đúng N thiếu** qua pipeline chuẩn `cdc.cmd.transmute` (OCC, idempotent).
3. **Audit rõ ràng (đáp F3)**: trả + log `{shadow_count, master_count_before, missing_found:[ids], dispatched, master_count_after}` ngay (đối soát được), + activity_log `recon-heal-b`. Operator thấy chính xác heal cái gì.
4. **MANUAL only**: KHÔNG đụng `runReconcileCycle` (không auto). Chỉ chạy khi gọi endpoint/bấm nút.

## Phạm vi file (dự kiến)
- `centralized-data-service/internal/handler/recon_heal_v4.go` — `healSegmentB` (robust diff + audit). Có thể +1 helper diff.
- (nếu cần) `reconciliation_handler_heal.go` (CMS) — surface audit trong response. Không schema, không auto.

## Verify (red→green)
- Reproduce: `export_jobs_mt` đang shadow 170 / master 163 (gap 7 thật).
- Bấm heal (manual) → log/response liệt kê đúng 7 id thiếu + dispatched 7 → sau vài giây master = 170 (đối soát count). Re-bấm → noop (0 thiếu).
- `go build` + restart worker + trigger thật.

## Quyết định kỹ thuật (đã verify)
- Worker `ReconHandler` có `shadowDB` (đọc shadow) nhưng **KHÔNG có handle đọc master/dest (5434)**. Diff tươi cross-DB ⇒ phải wire thêm `ReconCore.destAgent` vào handler = tăng impact + rủi ro.
- **Chọn (minimal-impact): full re-transmute** — `healSegmentB` publish `cdc.cmd.transmute {master_table}` (KHÔNG `_source_ids`) → `HandleTransmute` full scan shadow + **OCC upsert** (insert record thiếu, skip record master mới hơn) ⇒ **guaranteed gap-close**, không phụ thuộc recon report (đáp F1/F2).
- **Audit (đáp F3)**: `HandleTransmute` đã trả `scanned/inserted/updated` qua `cdc.result.transmute` + `transmute.completed`; **`inserted` = số record thiếu vừa heal**. Surface vào activity_log `recon-heal-b` + heal response `{status:"dispatched", mode:"full-retransmute"}`. Operator đối soát: master_before + inserted = master_after; re-bấm → inserted=0 (đã khớp).
- Đổi so với hiện tại: bỏ nhánh đọc report + map gpay→source + chunk (đang over-report cả bảng anyway) → thay bằng 1 publish full. **Code ÍT hơn, robust hơn, vẫn MANUAL.**
- Phạm vi thu hẹp: chỉ `recon_heal_v4.go healSegmentB` (+ optional surface stats). Không wire dest, không schema, không auto.
