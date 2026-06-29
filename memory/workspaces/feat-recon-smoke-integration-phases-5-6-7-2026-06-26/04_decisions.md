# Architectural Decisions: feat-recon-smoke-integration-phases-5-6-7-2026-06-26

## 1. Duy trì tính tương thích ngược của API đối soát (Backward Compatibility)
- **Quyết định**: Thay vì thay đổi toàn bộ API endpoints và phá vỡ cấu trúc DTO cũ, chúng tôi sử dụng SQL Raw SELECT kết hợp với các biểu thức `CASE WHEN` tại `recon_read_repo_gorm.go`.
- **Lý do**: Giúp frontend cũ vẫn hoạt động bình thường, đồng thời cung cấp đầy đủ thông tin 3 tầng mới (`source_total`, `source_active`, `shadow_total`, `shadow_active`, `master_total`, `master_active`) cho các thành phần UI mới.

## 2. Gom nhóm cột hiển thị (Column Grouping) trong Web UI
- **Quyết định**: Sử dụng tính năng Column Grouping của Ant Design Table trong `DataIntegrity.tsx` để gom các cột thành 3 cột lớn hoạt động song song (Source, Shadow, Master).
- **Lý do**: Nâng cao trải nghiệm người dùng (UX), giúp người vận hành (operator) dễ dàng đối chiếu dữ liệu giữa MongoDB, Postgres Shadow, và Postgres Master.

## 3. Prometheus Metrics O(1) và Ngăn ngừa Cardinality Explosion
- **Quyết định**: Định nghĩa 3 metrics Prometheus O(1) mới (`cdc_recon_smoke_latency`, `cdc_recon_smoke_status`, `cdc_recon_smoke_drift_count`) với nhãn cố định `["segment", "table"]`. Nhãn `table` chỉ lấy tên bảng đích (Target Table/Master Table).
- **Lý do**: Đảm bảo số lượng time-series sinh ra ở mức tối thiểu, ngăn ngừa rò rỉ bộ nhớ (memory leak) hoặc làm quá tải Prometheus TSDB của hệ thống SigNoz.
