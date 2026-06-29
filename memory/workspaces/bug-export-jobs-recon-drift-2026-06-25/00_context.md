# Context: Export Jobs Reconciliation Drift

## Vấn đề hiện tại
- **Mô tả**: Bản ghi đối soát (reconciliation report) cho bảng `export_jobs` báo trạng thái `drift` ở `segment_b_window` (ID 4024).
  - `SourceCount` (shadow active): 452
  - `DestCount` (master active): 0
  - `Diff`: 452
  - `MissingCount`: 457
  - `TotalSourceCount`: 457
  - `TotalDestCount`: 0
  - Đáng chú ý là bản ghi `count_total` (ID 4021) trước đó lại báo `ok` với cả `SourceCount` và `DestCount` đều bằng 452, `total_source_count` và `total_dest_count` đều bằng 457.
- **Mục tiêu**: Điều tra lý do tại sao ở `segment_b_window`, counts trên master table bị trả về `0`, dẫn đến báo drift 452 records, trong khi `count_total` lại khớp. Thiết kế và thực hiện giải pháp khắc phục.

## Các thành phần liên quan
1. **Recon Engine (centralized-data-service)**:
   - `internal/service/recon/recon_tier_b.go`
   - `internal/service/recon/recon_dest_query.go`
2. **Master Database (goopay_dest)**:
   - Cột `_source_ts` trong bảng `export_jobs`.
