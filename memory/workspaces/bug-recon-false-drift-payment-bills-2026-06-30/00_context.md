# Workspace Context: Fix False Drift on Recon payment_bills

## Problem Description
* Bảng `payment_bills` báo chênh lệch ảo 1.410 bản ghi khi đối soát Tier 1 (Mongo Source vs Postgres Shadow).
* **Nguyên nhân gốc rễ (Root Cause)**: Phía Postgres Shadow (Destination Agent) sử dụng metadata CDC timestamp `_source_ts` để lọc cửa sổ và hash dữ liệu, trong khi Mongo Source sử dụng domain timestamp (`lastUpdatedAt`). Gần đây, Debezium chạy backfill/snapshot khiến `_source_ts` của 1.410 bản ghi này bị ghi đè sang ngày 29/06/2026 (trong khi domain timestamp thực tế vẫn là 02/02/2026). Do chênh lệch hệ quy chiếu lọc này, phía Postgres query ra 1.410 bản ghi còn Mongo Source thì không, dẫn đến báo drift ảo.

## Solution
* Cập nhật `ReconDestAgent` để hỗ trợ lọc cửa sổ, group bucket và hash fingerprint theo cột domain timestamp (ví dụ: `lastUpdatedAt`) thay vì fix cứng metadata CDC timestamp `_source_ts`.
* Cập nhật `recon_tier_a.go` truyền `resolvedTS` (domain timestamp field) vào tất cả các phương thức của `ReconDestAgent`.
* Cập nhật `recon_tier_b.go` truyền `_source_ts` để tiếp tục so sánh CDC stream time cho đối soát Shadow vs Master.

## Definition of Done (DoD)
* Cả `ReconDestAgent` và các query/hash method của nó đều hỗ trợ dynamic timestamp field.
* `recon_tier_a.go` và `recon_tier_b.go` gọi `destAgent` với các timestamp field tương ứng.
* Unit test suite của `recon` pass 100%.
* Viết thêm unit tests cho `ReconDestAgent` hỗ trợ dynamic timestamp.
