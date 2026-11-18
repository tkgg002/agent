# L1 Base Rules — Lens SECURITY (`B-SEC-*`)

> Áp dụng cho mọi codebase, mọi ngôn ngữ. Mỗi mục là 1 câu hỏi có/không; trả lời "có vấn đề" **chỉ khi** có `REF: <path>:<start>-<end>` + trích code. Không có REF → không được tính là finding.
> Severity mặc định ghi trong ngoặc; Pack/Profile có thể nâng (không hạ) mức.

## 1. Secret & dữ liệu nhạy cảm
- **B-SEC-01 (S0)** — Diff có hardcode secret (API key, token, password, private key, connection string có mật khẩu, HMAC secret) trong code, config commit, test fixture hay comment không?
- **B-SEC-02 (S0)** — Diff có log/in/trả ra response bất kỳ secret hoặc dữ liệu nhận diện cá nhân (PII) ở dạng thô không? Có mask chuẩn chưa (`***`, chỉ giữ 4 ký tự cuối)?
- **B-SEC-03 (S1)** — Secret/PII có đi qua URL/query string, header không chuẩn, hay file tạm không được xoá không?
- **B-SEC-04 (S1)** — Error message trả về client có lộ stack trace, đường dẫn nội bộ, câu SQL, tên bảng/cột, phiên bản thư viện không?

## 2. Injection & đầu vào
- **B-SEC-05 (S0)** — Có nối chuỗi từ input ngoài vào SQL / NoSQL filter / shell command / template / LDAP / regex không? Phải dùng parameterized query / builder / allow-list.
- **B-SEC-06 (S1)** — Tên bảng, schema, cột, index **động** có được quote/allow-list trước khi ghép vào câu lệnh không?
- **B-SEC-07 (S1)** — Input từ API/consumer/webhook có được validate kiểu, độ dài, kích thước body, phần tử mảng tối đa trước khi xử lý không?
- **B-SEC-08 (S1)** — Deserialization (JSON/YAML/XML/BSON) có giới hạn kích thước/độ sâu không? Có nhánh parse "tin tưởng" dữ liệu ngoài không?
- **B-SEC-09 (S1)** — Path từ input có được chuẩn hoá và chặn traversal (`..`, symlink) không?

## 3. Xác thực & phân quyền
- **B-SEC-10 (S0)** — Endpoint/handler/consumer mới có đi qua lớp auth hiện hữu (middleware, interceptor, guard) không? Có route nào bypass không?
- **B-SEC-11 (S0)** — Truy cập tài nguyên theo id có kiểm tra **owner/tenant scope** không (IDOR)? Query theo id phải kèm điều kiện chủ sở hữu.
- **B-SEC-12 (S1)** — So sánh token/chữ ký/OTP có dùng constant-time compare không?
- **B-SEC-13 (S1)** — Có thay đổi quyền mặc định (default allow), nới CORS `*`, tắt CSRF, tắt TLS verify, hoặc hạ cấp cipher/hash không?

## 4. Crypto & ngẫu nhiên
- **B-SEC-14 (S1)** — Có dùng hash yếu (MD5/SHA1) cho chữ ký/password, random không phải CSPRNG cho token/OTP/nonce, hay tự chế thuật toán mã hoá không?
- **B-SEC-15 (S1)** — Key/IV/salt có bị tái sử dụng cứng hoặc hardcode không?

## 5. Chống lạm dụng & vận hành
- **B-SEC-16 (S1)** — Endpoint ghi/tốn tài nguyên/gửi OTP có rate limit / throttle / idempotency guard không?
- **B-SEC-17 (S2)** — Dependency mới thêm có nguồn tin cậy, phiên bản pin, không có CVE đã biết không? Lockfile có cập nhật không?
- **B-SEC-18 (S2)** — Có mở debug endpoint, flag `insecure`, verbose log mức production, hoặc để lại code "tạm" (TODO bypass) không?
- **B-SEC-19 (S1)** — Có DDL/migration/thay đổi quyền DB chạy trong luồng xử lý runtime (worker/handler) thay vì migration có kiểm soát không?

## 6. Memory & tài liệu (áp dụng khi diff chạm `agent/memory/` hoặc docs)
- **B-SEC-20 (S0)** — File memory/docs có chứa secret/PII thật chưa mask không?

## Lệnh hỗ trợ quét nhanh (0 LLM token)
```bash
git diff <base>..<head> | grep -nE "(AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|ghp_[A-Za-z0-9]{30,}|-----BEGIN [A-Z ]*PRIVATE KEY|(mongodb|postgres|postgresql|mysql|redis|amqp)://[^*\s:/]+:[^*\s@]+@|(password|passwd|secret|token|api[_-]?key)\s*[:=]\s*['\"][^'\"]{6,})"
```
