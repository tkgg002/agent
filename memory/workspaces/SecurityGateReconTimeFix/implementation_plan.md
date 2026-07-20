# Kế hoạch triển khai chi tiết của AI - Security Gate Recon Time Zone Fix

## 1. Bối cảnh
Người dùng yêu cầu thực hiện quy trình `/security-agent` để rà soát bảo mật đối với các thay đổi xử lý múi giờ trên các file:
- `internal/service/recon/recon_stream.go`
- `internal/service/recon/recon_query.go`
- `internal/service/recon/recon_dest_hash.go`
- `internal/service/recon/recon_postgres_source_test.go`

## 2. Kế hoạch chi tiết
- **Bước 1:** Rà soát tĩnh các thay đổi trong file `recon_dest_hash.go` và `recon_query.go` để phát hiện rủi ro SQL Injection hoặc sai sót logic múi giờ.
- **Bước 2:** Rà soát tĩnh các thay đổi trong file `recon_stream.go` liên quan đến `resolvePostgresTimeParams` và format timestamp thô.
- **Bước 3:** Đánh giá rò rỉ Secrets/PII trong các thay đổi và code test.
- **Bước 4:** Ghi chép phân tích bảo mật vào `13_analysis_time_fix.md`.
- **Bước 5:** Tổng hợp thay đổi vào `11_report_time_fix.md`.
- **Bước 6:** Chạy script linter quy trình `python3 agent/tooling/verify_governance.py`.
- **Bước 7:** Báo cáo kết quả chi tiết về cho parent agent.
