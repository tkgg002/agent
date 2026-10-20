---
name: security-agent
description: >-
  Security Gate specialist workflow — scans code diffs and workspace memory files for
  hardcoded secrets, PII leaks (Rule #20), injection vulnerabilities, broken authz,
  and unsafe dependencies before task completion. Enforces Rule #15 (No git push).
---

# Security Agent Skill (Security Gate & Secret/PII Guard)

> **Quy tắc cốt lõi:** Rule #3 (Security Auto-Check), Rule #14-G7 (Adversarial Review), Rule #15 (Security Gate & No Push), Rule #20 (No-Secret-in-Memory).

## Khi nào sử dụng
- Bắt buộc chạy ở chặng **S5 SELF-AUDIT** cho mọi task có sửa đổi source code, cấu hình, hoặc tài liệu memory.
- Khi User yêu cầu rà soát bảo mật (`/security-agent`).

## Quy trình Kiểm định 5 Tầng & Tiếp nhận Báo cáo PentAGI

### 1. Quét Secret & PII trong Code và Memory (Rule #20 — Bắt buộc 100%)
Quét toàn bộ `git diff` và các file mới/sửa trong `agent/memory/workspaces/<ws>/`:
```bash
grep -rnE "(AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|ATATT[A-Za-z0-9_-]{20,}|ghp_[A-Za-z0-9]{30,}|Bearer [A-Za-z0-9._-]{20,}|(mongodb|postgres|postgresql|mysql|redis)://[^*\s:/]+:[^*\s@]+@|password\s*[:=]\s*[^\s*'\"]{6,})" <paths>
```
- **Trong `agent/memory/`:** Tuyệt đối không được chứa password, token, API key, private key, URI có mật khẩu, hoặc PII thật (số điện thoại, email thật, CCCD, số tài khoản). Phải mask thành `mongodb://***:***@host/db`, `Bearer ***`, `phone=09xx***`.
- **Trong Source Code:** Mọi khóa bí mật phải đọc từ biến môi trường / secret manager; `.env` chứa secret thật phải nằm trong `.gitignore`.

### 2. Kiểm tra An toàn Đầu vào & Truy vấn (Input & Query Safety)
- **Injection:** Không nối chuỗi trực tiếp từ input người dùng vào câu lệnh SQL, NoSQL filter, shell command (`os/exec`, `subprocess` với `shell=True` nhận input ngoài), hoặc template HTML.
- **Validation:** Mọi payload từ API/Consumer/Webhook phải được kiểm tra kiểu dữ liệu, giới hạn độ dài/kích thước trước khi xử lý.

### 3. Kiểm tra Xác thực, Phân quyền & Lộ Thông tin (AuthN/AuthZ & Exposure)
- Các endpoint/command mới có gắn đúng middleware xác thực và kiểm tra quyền (RBAC/tenant isolation) chưa?
- Log và thông báo lỗi trả về client có vô tình làm lộ stack trace nội bộ, cấu trúc bảng DB, hoặc token không?

### 4. Kiểm tra Kỷ luật VCS (Rule #15 & #19)
- Xác nhận không có file nhạy cảm (`.env`, dump DB, private key) nằm trong `git status` staged.
- Nhắc nhở và cưỡng chế: **TUYỆT ĐỐI KHÔNG chạy `git push`** — chỉ tạo local commit restore-point, quyền push thuộc về User.

### 5. Kiểm thử Xâm nhập Động (Layer 5: Dynamic Pentest Gate via `pentagi_bridge.py`)
- **Chặng S5 (Inline Scoped DAST):** Khi component trong `<project-root>/.agent-project.json` có khai báo `pentest_check` (hoặc thay đổi chạm tới HTTP/gRPC/GraphQL handler, AuthN/AuthZ, SQL/NoSQL query), kích hoạt kiểm thử xâm nhập động có phạm vi qua `pentagi_bridge.py`:
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/pentagi_bridge.py trigger \
    --target <local_or_whitelisted_url> --project-root <project_root> \
    --prompt "Scoped DAST check on modified endpoints"
  ```
  - **Cổng chặn an toàn (Target Whitelist Guard):** `pentagi_bridge.py` tự động đối chiếu `--target` với `allowed_pentest_targets` trong `.agent-project.json` (mặc định chỉ cho phép `localhost` / `127.0.0.1` / `[::1]`). Mọi target ngoài danh sách cho phép sẽ bị chặn cứng (`exit code 2`).
- **Chặng S1 (Nightly Pentest Report Intake):** Khi tiếp nhận báo cáo pentest ban đêm từ PentAGI, **CẤM sửa code trực tiếp theo báo cáo**. Bắt buộc chuyển đổi báo cáo sang tài liệu `S1 Intake` (tự động mask mọi secret/credential lọt trong exploit dump theo Rule #20):
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/pentagi_bridge.py intake \
    --report <path_to_pentagi_report.md> --workspace <ws_name> --suffix <suffix>
  ```
  Sau khi tạo bộ `01_requirements_<suffix>.md`, `05_progress_<suffix>.md`, `08_tasks_<suffix>.md`, bắt buộc đi qua `S2 Reproduce (Gate G2)` → `S3 Solution (DG1/DG2)` → **chờ User `APPROVE`** trước khi sửa code ở `S4`.

## Mẫu Báo cáo Security Gate (Ghi vào `06_validation_*.md` hoặc `11_report_*.md`)
```markdown
### Security Gate Report
| Hạng mục | Kết quả | Chi tiết |
|----------|---------|----------|
| Secret & PII Scan (Code + Memory) | PASS / FAIL | Không phát hiện secret/PII thô |
| Input & Query Injection Check | PASS / FAIL | <ghi chú> |
| AuthN / AuthZ & Error Exposure | PASS / FAIL | <ghi chú> |
| VCS Hygiene (No secret staged, No push) | PASS / FAIL | <ghi chú> |
| Dynamic Pentest Gate (DAST / PentAGI) | PASS / N/A / FAIL | <kết quả quét hoặc N/A nếu không có API endpoint> |
- **Verdict:** ✅ PASS / ❌ BLOCK
```
