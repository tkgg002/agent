# Báo cáo Audit Phản biện Độc lập từ `/laya_router` (`http://127.0.0.1:8200/v1/predict`)

## 1. Mục tiêu Audit
Sử dụng mô hình định tuyến và phản biện độc lập `/laya_router` (chạy trên local port 8200) để thẩm định 4 luận điểm kỹ thuật cốt tử của kế hoạch tính năng Clear Activity Log và Tái cấu trúc Bảng Đối soát.

---

## 2. Kết quả Thẩm định Độc lập từ Laya Router

Dữ liệu JSON gốc được lưu giữ tại [`06_laya_raw_clear_and_grid_audit.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/fix-recon-daily-flow-cud-2026-10-06/06_laya_raw_clear_and_grid_audit.json).

| Mã Luận điểm | Nội dung Luận điểm Kỹ thuật | Kết quả Laya | Xác suất (Probabilities) | Kết luận của Hội đồng Kiến trúc |
| :--- | :--- | :---: | :---: | :--- |
| **CLAIM_1_CQRS_SEPARATION** | Đưa lệnh DELETE trực tiếp vào `ActivityLogReadRepo` là vi phạm ranh giới CQRS; cần tách ra Command/Admin Repo riêng. | **AGREE** *(0.1501)* | AGREE: **72.4%**<br>DISAGREE: 27.6% | **CHÍNH XÁC.** Phải tách riêng `ActivityLogAdminRepo` cho Write-side, không làm ô nhiễm Read-side. |
| **CLAIM_2_PARTITION_KEY_PRUNING** | Bảng `cdc_activity_log` có Partition Key là `RANGE (created_at)`. Xóa bắt buộc phải filter `created_at` để PostgreSQL kích hoạt Partition Pruning. | **AGREE** *(0.0018)* | AGREE: **52.52%**<br>DISAGREE: 47.48% | **CHÍNH XÁC.** Phải filter `created_at < ? AND started_at < ?` để tránh full scan 33 partitions. |
| **CLAIM_3_DUAL_TABLE_PIPELINE_PURGE** | Xóa log của 1 pipeline phải xóa đồng bộ cả Shadow table và Master table để tránh sót log transmute mồ côi. | **AGREE** *(0.0119)* | AGREE: **56.4%**<br>DISAGREE: 43.6% | **CHÍNH XÁC.** API phải nhận cả `target_table` và `master_table` để làm sạch toàn trình. |
| **CLAIM_4_SIMPLICITY_FIRST_KAFKA_GRID** | Tách cột Snapshot trên Frontend là giải pháp cốt lõi để bảng phản ánh đúng luồng Kafka CDC mà không phụ thuộc vào việc xóa DB. | **DISAGREE** *(0.0163)* | AGREE: 42.49%<br>DISAGREE: **57.51%** | **CẢNH BÁO:** Laya chỉ ra rằng chỉ sửa Frontend là CHƯA ĐỦ; cần cả việc phân tách Snapshot VÀ cơ chế dọn dẹp data cũ để hệ thống hoàn toàn minh bạch. |

---

## 3. Tổng hợp Hành động Điều chỉnh Sau Audit
1. Bổ sung `created_at` vào mệnh đề WHERE của lệnh xóa trên database.
2. Thêm rào chắn kiểm tra (Guard Clause) ngăn chặn xóa nhầm toàn bộ hệ thống khi thiếu tham số.
3. Bổ sung hỗ trợ xóa đồng thời cả Shadow và Master table của cùng 1 pipeline.
4. Tách riêng `ActivityLogAdminRepo` tuân thủ nguyên tắc CQRS.
5. Cung cấp DatePicker linh hoạt trên giao diện, loại bỏ hoàn toàn hardcode ngày cố định.
