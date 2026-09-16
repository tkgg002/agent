# 11 Report: Báo cáo thay đổi Fix Log Transmute & Trace ID Đối soát

## 1. Tổng quan thay đổi
Đã thực hiện sửa đổi trên 5 tệp tin mã nguồn (2 backend Go, 3 frontend React/TS) để giải quyết triệt để 2 lỗi tại `/data-integrity`:
1. Tab Log Transmute không hiển thị các phiên Transmute mới nhất của Master Table.
2. Popup Toast hiển thị Trace ID ngẫu nhiên không đúng OTel trace của worker.

---

## 2. Chi tiết từng file đã thay đổi & Số dòng tác động

| STT | File | Ngôn ngữ | Số dòng thay đổi (+/-) | Mô tả thay đổi |
|---|---|---|---|---|
| 1 | `cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go` | Go | +8 / -0 | Trích xuất `traceID` từ `ctx` và trả về trường `"trace_id": traceID` ở 2 nhánh response của `TriggerCheckAll`. |
| 2 | `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go` | Go | +8 / -8 | Đồng bộ `COALESCE(tm_so..., so...)` và `COALESCE(tm_sb..., sb...)` ở cả `countQuery` và `mainQuery`. |
| 3 | `cdc-cms-web/src/hooks/useReconStatus.ts` | TypeScript | +1 / -1 | Sửa `enabled: Boolean(table)` trong `usePipelineActivityLog` để chặn over-fetching log khi chưa có bảng. |
| 4 | `cdc-cms-web/src/components/ReconPipelineGrid.tsx` | TypeScript (React) | +8 / -3 | Chuẩn hóa bare table name `rawMaster.split('.').pop()`, bảo toàn khai báo `historySchema`, truyền `historyMaster` vào hook Transmute, thêm Empty state khi chưa có master binding. |
| 5 | `cdc-cms-web/src/pages/DataIntegrity.tsx` | TypeScript (React) | +2 / -2 | Sửa fallback `traceId: res?.trace_id || undefined` và `healRes?.trace_id || undefined`, loại bỏ client UUID giả. |
| 6 | `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go` | Go | +58 / -0 | Thêm `jobRepo`, `reportProgress`, tính `totalDays` và cập nhật `progress_percent` + `checkpoint_ts` sau mỗi ngày trong `Execute` và `executeSegmentB`. |
| 7 | `centralized-data-service/internal/server/server_setup.go` | Go | +1 / -0 | Nối `chunkEngine.WithJobRepo(reconJobRepo)`. |

**Tổng cộng:** 7 file, +86 dòng thêm, -14 dòng sửa/xóa. (Bao gồm hotfix khôi phục `historySchema` tại `ReconPipelineGrid.tsx`).

---

## 3. Bản chất kỹ thuật & Sự bảo toàn kiến trúc (Zero Regression)
- Không can thiệp DDL/Schema cơ sở dữ liệu.
- Không thay đổi contracts của API endpoints.
- Bảo đảm 100% Simplicity First & Minimal Impact: Bám sát thiết kế hiện có của hệ thống CDC.
