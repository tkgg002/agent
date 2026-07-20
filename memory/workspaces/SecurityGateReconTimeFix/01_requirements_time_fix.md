# Yêu cầu Rà soát Bảo mật (Security Gate) - Recon Time Zone Fix

## 1. Mục tiêu
Thực hiện Security Gate audit đối với các thay đổi liên quan đến xử lý múi giờ trên các file:
- `internal/service/recon/recon_stream.go`
- `internal/service/recon/recon_query.go`
- `internal/service/recon/recon_dest_hash.go`
- `internal/service/recon/recon_postgres_source_test.go`

## 2. Quy trình đối chiếu
Bám sát quy trình `/security-agent` trong `GEMINI.md`:
- **Code Review & Input Validation:** Kiểm tra tham số đầu vào, phòng chống SQL Injection (đặc biệt là Raw SQL).
- **Secrets Check:** Đảm bảo không rò rỉ credential, API keys, password, token.
- **PII Leakage Check:** Đảm bảo không lộ thông tin cá nhân của khách hàng (email, SĐT, số tài khoản, CCCD...).
- **Verdict & Report:** Xuất báo cáo bảo mật theo đúng mẫu quy định.
