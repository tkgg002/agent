# report_recon_dashboard_pipeline_grid_2026-06-10.md — Dashboard Pipeline Grid (data-lineage)

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | Bổ sung sau Recon V4 P0-P4

## 1. Đã làm gì
Dashboard 2 tầng theo trục **data-lineage** (blueprint Boss chốt: "đi trực tiếp từ trục dữ liệu, không KPIs chung chung") — **thuần FE, 0 dòng BE đổi** (tận dụng trọn dữ liệu Recon V4 đã xây):

### Tầng 1 — Master Pipeline Grid (tab "Pipelines" — tab MẶC ĐỊNH mới của Data Integrity)
- Mỗi dòng = 1 pipeline `Source → Shadow → Master`, ghép từ report **Segment A** (source↔shadow) + **Segment B** (shadow↔master) theo lineage key (`B.source_db` shadow FQN ↔ `A.target_table`); bảng chưa có master binding vẫn hiện (master = "—").
- Cột: Pipeline (3 trạm + mũi tên) · Source/Shadow/Master (recs, window 7d) · **Drift 2 chặng** (`ingest: shadow−source`, `transmute: master−shadow`; âm=thiếu đỏ, dương=thừa/orphan vàng — đúng convention blueprint) · Recon cuối lúc · Trạng thái tổng (`Khớp/Lệch/Lagging/Cảnh báo` — Lagging khi lag>30').

### Tầng 2 — Drill-down Drawer (click dòng)
- **A. Flow map**: 3 trạm (counts) + 2 chặng (`Debezium CDC` lag ingest, `Transmute Worker` lag transmute — màu theo mức).
- **B. Convergence chart** (recharts có sẵn): line Source/Shadow/Master theo từng phiên recon — các đường bám nhau = không lag; trạm sau rớt = tắc ở chặng đó.
- **C. Nhật ký đối soát** 30 phiên gần nhất của riêng pipeline: phiên lúc / loại scan (Level 1/2 · Segment B · Deep) / KHỚP-LỆCH / chi tiết (counts, thiếu, stale, đã heal).
- Nguồn data: `GET /api/reconciliation/report/:table` (**TableHistory — endpoint có sẵn nhưng FE chưa từng dùng → đóng GAP4** từ khảo sát đầu).

## 2. Files THỰC TẾ đã sửa (git diff — cdc-cms-web only)
| File | Thay đổi |
|---|---|
| `src/components/ReconPipelineGrid.tsx` | **NEW 324 dòng** — buildPipelines (ghép lineage A+B), grid, drawer (FlowStation/FlowEdge/DrillDown), chart, log table |
| `src/hooks/useReconStatus.ts` | +29/− — `useTableHistory` hook + type `field_diffs` |
| `src/pages/DataIntegrity.tsx` | +tab `Pipelines` (default) + import (~+14 trong stat 63 — phần còn lại là cột Segment/Lag của P4) |
*(MasterRegistry.tsx +70 trong stat thuộc task trước, không liên quan.)*
**BE: 0 file đổi.**

## 3. Verify
| Bước | Kết quả |
|------|---------|
| `npm run build` | ✅ PASS (781ms; fix 1 lỗi type `field_diffs` trong quá trình build) |
| Data-shape backing | ✅ đã verify ở P3: query per-(table,segment)+JOIN lag trả 12 rows — đúng input buildPipelines |
| Services | ✅ 3/3 LISTEN (worker 8082, cms 8083, FE 5173 — dev server tự reload source mới) |
| Browser check | ⏳ Boss mở `localhost:5173/...` trang Data Integrity → tab **Pipelines** (mặc định): grid lineage; click dòng `b3`/`export_jobs_mt` → drawer flow-map + chart + nhật ký |

## 4. Khác blueprint (chủ đích, ghi rõ)
- Counts hiển thị theo **window đối soát 7d** (đúng dữ liệu recon đo) thay vì full-table count — full count nằm sẵn ở tab Tổng quan (`full_source_count`); tránh thêm COUNT(*) full-table tốn kém mỗi lần render.
- "Auto-Heal Success %" (Row 1 blueprint Grafana) chưa có — cần aggregate healed/dispatched theo 24h; để vòng sau nếu Boss cần (đã có `healed_count` per report làm nguyên liệu).
- Convergence chart lấy điểm theo **phiên recon** (không phải continuous time-series) — đúng nguồn dữ liệu hiện có, không bịa metric.
