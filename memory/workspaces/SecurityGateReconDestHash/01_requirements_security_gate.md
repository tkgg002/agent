# Yêu cầu Rà soát Bảo mật (Security Gate) - Recon Dest Hash

## 1. Mục tiêu
Thực hiện Security Gate audit đối với các thay đổi trên hai file:
- `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go`
- `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go`

## 2. Quy trình đối chiếu
Bám sát quy trình `/security-agent` trong `GEMINI.md`:
- **Code Review & Input Validation:** Kiểm tra tham số đầu vào, phòng chống SQL Injection (đặc biệt là Raw SQL).
- **Secrets Check:** Đảm bảo không rò rỉ credential, API keys, password, token.
- **PII Leakage Check:** Đảm bảo không lộ thông tin cá nhân của khách hàng (email, SĐT, số tài khoản, CCCD...).
- **Verdict & Report:** Xuất báo cáo bảo mật theo đúng mẫu quy định.
