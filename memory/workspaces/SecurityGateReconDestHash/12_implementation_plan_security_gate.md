# Kế hoạch triển khai chi tiết của AI - Security Gate Recon Dest Hash

## 1. Bối cảnh
Người dùng yêu cầu thực hiện quy trình `/security-agent` để rà soát bảo mật đối với các thay đổi trên:
- `recon_dest_hash.go`
- `recon_dest_agent_test.go`

## 2. Kế hoạch chi tiết
- **Bước 1:** Đọc nội dung file `recon_dest_hash.go` bằng tool `view_file` để nắm cấu trúc hiện tại và kiểm tra SQL Injection, PII, Secrets.
- **Bước 2:** Đọc nội dung file `recon_dest_agent_test.go` bằng tool `view_file` tương tự để rà soát.
- **Bước 3:** Sử dụng `git diff` thông qua `run_command` nếu cần thiết hoặc rà soát toàn bộ file để xác định các thay đổi gần đây.
- **Bước 4:** Lập báo cáo lỗi bảo mật theo đúng mẫu `/security-agent` (nhóm theo 4 category: Input Validation, Secrets, Dependencies, API Security).
- **Bước 5:** Thực thi script linter quy trình `python3 agent/tooling/verify_governance.py`.
- **Bước 6:** Gửi message báo cáo cho parent agent và trả lời user bằng tiếng Việt.
