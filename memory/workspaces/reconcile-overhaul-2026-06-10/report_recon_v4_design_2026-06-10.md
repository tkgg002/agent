# report_recon_v4_design_2026-06-10.md — Thiết kế tổng thể Reconcile V4

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | Workspace `reconcile-overhaul-2026-06-10`
> **Loại turn**: DESIGN ONLY — đáp ứng yêu cầu "làm giải pháp tổng thể"; **0 file source code thay đổi** trong turn này.

## 1. Bối cảnh & sửa sai
Turn trước tôi sai deliverable: Boss yêu cầu giải pháp tổng thể, tôi đi vá code cục bộ. Đã DỪNG theo Rule 7, ghi lesson `[2026-06-10] Verb "làm giải pháp tổng thể" = deliverable là DESIGN...` vào `lessons.md` (nhóm 1 Process & Governance, 63→64 pattern), rồi làm lại đúng đề bài.

## 2. Deliverables turn này (files tạo/đổi — đều là docs)
| File | Loại | Nội dung |
|------|------|----------|
| `agent/memory/global/lessons.md` | APPEND lesson | deliverable-mismatch / strategic-vs-tactical |
| `10_gap_analysis_recon_v4.md` | NEW (~40 dòng) | Gap matrix hiện trạng vs 4 trụ chuẩn; phân loại GIỮ / BỎ-LÀM-LẠI / XÂY MỚI |
| `09_tasks_solution_recon_v4.md` | NEW (~170 dòng) | Thiết kế đích V4: 2-segment E2E, watermark adaptive, heal re-trigger, alert+lag, code demo SQL/Go/payload, roadmap P1-P4 (~7d), DoD từng phase |
| `05_progress.md` | APPEND | log turn |
| report này | NEW | — |

**Source code: 0 file đổi** (`git status` worker/cms/web không thêm diff mới so với cuối turn trước). Verify kỹ thuật cho thiết kế thực hiện bằng psql read-only (cột `_gpay_id/_source_ts` trên 5436+5434, transmuter conflict key) — không ghi gì vào DB.

## 3. Tóm tắt thiết kế (chi tiết trong 09)
- **E2E = Segment A (source↔shadow, GIỮ engine 3-tier vừa hồi sinh) ∧ Segment B (shadow↔master, XÂY MỚI)** — so theo `(_gpay_id, _source_ts)` (đã verify cột thật 2 DB); lỗi tự định vị tắc ở ingest hay transmute.
- **Watermark adaptive**: freeze margin = f(ingest_lag + transmute_lag), kẹp [5', 60'] — hết false-positive khi lag cao.
- **Self-healing làm lại theo re-trigger**: A → Debezium incremental snapshot signal (handler đã có, KHÔNG dummy-update vào source DB); B → `cdc.cmd.transmute SourceIDs` (field đã có); ngưỡng an toàn >5000 ID / >5% → chỉ alert chờ operator. **Bỏ heal bypass** (đang disabled + đi tắt qua masking/mapping).
- **Alert** ngưỡng drift_pct → `cdc.evt.alert` (alert_manager cms đã có). **Lag monitoring** 3 điểm/bảng (ingest, transmute, backlog) vào bảng `recon_lag` + UI.
- **Nhịp chạy (chốt)**: hybrid — L1 10' liên tục, L2 6h, L3 off-peak; Segment B thêm event-driven sau `transmute.completed`.
- **Roadmap**: P1 SegmentB → P2 Heal re-trigger → P3 Watermark+Lag → P4 Alert+RowDiff+FE ≈ **7 ngày**, mỗi phase có DoD evidence.

## 4. Trạng thái services (kiểm trước khi báo done)
- Worker PID 9699 (`/tmp/cdc-worker-recon`) RUNNING — port 8082 LISTEN; cms 8083 RUNNING; FE 5173. Không service nào bị đổi trong turn này.

## 5. Chờ Boss
`approve recon v4` (Muscle thực thi P1) | `revise <điểm>`.
