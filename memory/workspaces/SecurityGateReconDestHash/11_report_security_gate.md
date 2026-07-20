# Báo cáo thay đổi & Kết quả Rà soát Bảo mật - Security Gate Recon Dest Hash

## 1. Overview Thay đổi
Đã thực hiện rà soát bảo mật cho 2 file:

### File 1: `internal/service/recon/recon_dest_hash.go`
- **Số lượng dòng thay đổi:** +7 dòng, -2 dòng.
- **Chi tiết thay đổi:**
  - Bổ sung điều kiện loại trừ các bản ghi đã xóa `AND NOT "_deleted"` vào câu truy vấn SQL trong hàm `HashWindow` (áp dụng cho cả 2 nhánh: _source_ts mặc định và domain timestamp).
  - Bổ sung `AND NOT "_deleted"` vào câu truy vấn SQL trong hàm `BucketHash` (áp dụng cho cả 2 trường hợp: pagination từ đầu và pagination tiếp nối từ `lastID`).

### File 2: `internal/service/recon/recon_dest_agent_test.go`
- **Số lượng dòng thay đổi:** +6 dòng, -4 dòng.
- **Chi tiết thay đổi:**
  - Cập nhật các mock query expectations tương ứng trong các test case `TestDestAgent_HashWindow_DomainTS`, `TestDestAgent_BucketHash_DomainTS`, `TestDestAgent_HashWindow_Default`, `TestDestAgent_BucketHash_Default` để khớp với câu truy vấn mới có bổ sung `AND NOT "_deleted"`.

## 2. Kết quả Rà soát Bảo mật
- **Secrets Check:** 🟢 PASS (Không phát hiện hardcoded credentials).
- **PII Check:** 🟢 PASS (Không có PII rò rỉ; dữ liệu được băm trực tiếp qua MD5 và tổng hợp XOR).
- **Input Validation & SQL Injection:** 🟢 PASS (Tất cả định danh bảng/cột đều được validate bằng `validateIdent` và quote bằng `quoteIdent`/`quoteRelation`; dữ liệu động bind bằng parameter placeholder `?`).
- **Verdict:** ✅ **PASS**
