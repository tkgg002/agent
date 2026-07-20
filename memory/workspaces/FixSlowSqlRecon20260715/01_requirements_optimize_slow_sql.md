# Yêu cầu Tối ưu hóa Hiệu năng Truy vấn SQL Đối soát (Reconciliation Report & Failed Sync Logs)

## 1. Bối cảnh & Vấn đề
- Hệ thống phát sinh cảnh báo **SLOW SQL (>= 200ms)** đối với các truy vấn liên quan đến đối soát dữ liệu và nhật ký đồng bộ lỗi.
- **Vấn đề 1 (Reconciliation Report):** Truy vấn lấy danh sách báo cáo đối soát mới nhất (`ListLatest` trong `recon_read_repo_gorm.go`) tốn thời gian thực thi rất lớn (> 1.2s). Nguyên nhân là do cấu trúc truy vấn hiện tại thực hiện `LEFT JOIN LATERAL` với bảng `cdc_recon_smoke_result` trên toàn bộ lịch sử của bảng `cdc_reconciliation_report` trước khi áp dụng `DISTINCT ON` để lọc ra các dòng mới nhất. Điều này dẫn đến hàng chục nghìn lượt quét bảng phụ không cần thiết.
- **Vấn đề 2 (Failed Sync Logs Count):** Truy vấn đếm số lượng log đồng bộ lỗi (`Count` trong `ListFailedLogs` trong `recon_read_repo_gorm.go`) tốn thời gian ~240ms+. Nguyên nhân là do truy vấn đếm sử dụng lại toàn bộ cấu trúc query cơ sở `failedLogsBase` vốn chứa 2 `LEFT JOIN LATERAL` phức tạp với bảng `shadow_binding` và `source_object_registry`. Vì đây chỉ là query đếm số lượng bản ghi sau lọc (phục vụ phân trang), các phép JOIN này là hoàn toàn dư thừa vì chúng không ảnh hưởng đến số lượng dòng trả về (đều là `LEFT JOIN` không lọc dòng).

## 2. Mục tiêu kỹ thuật
- Tối ưu hóa truy vấn `ListLatest` giảm thời gian thực thi xuống dưới **100ms** (mục tiêu thực tế là dưới **20ms** ở môi trường local).
- Tối ưu hóa truy vấn đếm dòng của `ListFailedLogs` loại bỏ hoàn toàn các phép JOIN dư thừa, giảm thời gian xuống dưới **20ms**.
- Đảm bảo độ chính xác dữ liệu 100%, không phá vỡ logic cũ, tương thích hoàn toàn cấu trúc kết quả trả về (`LatestReportRow` và `FailedLogRow`).
- Không gây regression trên các luồng nghiệp vụ khác.
