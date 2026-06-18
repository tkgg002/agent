# report_fix_dashboard_total_counts_2026-06-10.md — Fix 3 cột recs Pipeline Grid

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | User report: "Source/Shadow/Master (recs) đang không đúng"

## 1. Root cause
`buildPipelines` lấy `source_count/dest_count` từ report = số record **trong window đối soát 7d** (đúng nghĩa recon, sai nghĩa dashboard). Ví dụ thật: `b3` hiện Master=11 trong khi bảng thật 464 rows. Blueprint yêu cầu **tổng record tại thời điểm chạy recon gần nhất**.

## 2. Fix (tận gốc — recon đo thêm tổng thật mỗi vòng, không cheat FE)
- Migration **084**: report +`total_source_count` +`total_dest_count` BIGINT.
- Worker `RunTier1` (Segment A): +`sourceAgent.CountDocuments` (Mongo full, hàm CÓ SẴN) + `destAgent.CountRows` (shadow exact) → ghi totals; lỗi đếm → NULL, không fail run.
- Worker `RunSegmentB`: +`CountRows` shadow + master (PG exact ×2) → ghi totals.
- Model worker + cms: +2 fields (scan `r.*`).
- FE: type +2 fields; `buildPipelines` ưu tiên totals (fallback window-count cho report cũ chưa có); `driftAB/driftBC` tính từ totals (shadow−source / master−shadow); sửa label drawer "tổng record thật tại thời điểm recon gần nhất".

## 3. Files đã sửa (git)
| File | Đổi |
|---|---|
| cms `migrations/.../084_recon_total_counts.sql` | NEW |
| worker `internal/model/reconciliation_report.go` | +4 |
| worker `internal/service/recon_core.go` | +2 đoạn đo totals (Tier1 + SegB, ~+24) |
| cms `internal/model/reconciliation_report.go` | +3 |
| web `src/hooks/useReconStatus.ts` | +3 |
| web `src/components/ReconPipelineGrid.tsx` | buildPipelines totals + label (~+20/−10) |

## 4. Verify (đối chiếu TAY với DB thật — không chế số)
| Pipeline | Report totals | COUNT(*) tay | Khớp |
|---|---|---|---|
| `b3` (Segment B) | shadow **465** / master **464** | shadow_aaaaa.export_jobs_4 = **465**; dw….b3 = **464** | ✅ tuyệt đối |
| `export_jobs` (Segment A) | source **168** / shadow **170** | shadow_dev000.export_jobs = **170** | ✅ |
- Build worker + cms + FE: PASS; migration 084 applied; worker restart (binary p4b); services 3/3 LISTEN.
- **Giá trị lộ ra ngay**: b3 465 vs 464 = thiếu 1 row thật NGOÀI window 7d (window-based không thấy); export_jobs 168 vs 170 = shadow giữ 2 row source đã xoá — dashboard giờ phản ánh đúng thực trạng tổng thể.

## 5. Note chi phí
Mỗi vòng recon thêm: A = 1 Mongo CountDocuments + 1 PG COUNT; B = 2 PG COUNT — mỗi 30'/bảng, chấp nhận được. Bảng prod rất lớn nếu COUNT chậm → cân nhắc `pg_class.reltuples` estimate (để ngỏ, chưa over-engineer).
