# Report - Recon Source Agent Refactor

Tài liệu báo cáo chi tiết kết quả thực hiện phân rã file lớn `recon_source_agent.go` (1166 dòng ban đầu) thành các cấu phần chuyên biệt hơn.

## 1. Danh sách các file thay đổi / tạo mới
Thay đổi được áp dụng hoàn toàn trong package `recon` tại thư mục `internal/service/recon/`:

| File | Trạng thái | Mô tả |
|------|------------|-------|
| `recon_source_agent.go` | **MODIFY** | Chỉ giữ lại cấu trúc lõi `ReconSourceAgent`, constructor và logic quản lý kết nối MongoDB, circuit breaker. |
| `recon_models.go` | **NEW** | Chứa các struct định nghĩa dữ liệu (`ChunkHash`, `WindowResult`, `BucketHashResult`, `ReconSourceAgentConfig`), hằng số mã lỗi, và hàm phân loại lỗi Mongo. |
| `recon_hash.go` | **NEW** | Chứa logic băm XOR, xxhash và helper trích xuất ID/Datetime từ MongoDB. |
| `recon_query.go` | **NEW** | Chứa logic đếm số lượng tài liệu, aggregate count theo bucket và retry query tự động khi gặp lỗi transient. |
| `recon_stream.go` | **NEW** | Chứa logic stream dữ liệu keyset pagination (`$gt` trên `_id`) để chống cursor timeout và OOM. |
| `recon_legacy.go` | **NEW** | Chứa các legacy shims tương thích ngược (`GetChunkHashes`, `buildLegacyChunkHash`, `redactURL`). |

---

## 2. Thống kê số lượng dòng code thay đổi (Lines of Code)

*   **Số lượng dòng ban đầu**: **1166 lines**
*   **Số lượng dòng sau refactor**:

| File | Số dòng (LoC) |
|------|--------------|
| `recon_source_agent.go` (rút gọn) | 134 |
| `recon_models.go` (mới) | 121 |
| `recon_hash.go` (mới) | 202 |
| `recon_query.go` (mới) | 316 |
| `recon_stream.go` (mới) | 195 |
| `recon_legacy.go` (mới) | 55 |
| **Tổng cộng** | **1023 lines** |

> [!TIP]
> File core `recon_source_agent.go` giảm từ **1166 dòng xuống còn 134 dòng** (giảm ~88.5%).
> Tổng số dòng code giảm đi 143 dòng nhờ loại bỏ import trùng lặp, tối ưu hóa cấu trúc và loại bỏ logic dư thừa.

---

## 3. Kết quả xác minh (Verification Results)
1.  **Biên dịch**: Lệnh `go build ./...` hoàn thành thành công 100% không có lỗi.
2.  **Unit Tests**:
    *   Chạy unit tests package `recon` (`go test -v ./internal/service/recon/...`): **PASS**
    *   Chạy toàn bộ unit tests dự án (`go test ./...`): **PASS 100%** (bao gồm cả các integration test).
3.  **Bảo mật**: Đã chạy rà soát và tạo báo cáo bảo mật tại `report_security_recon_refactor.md` đạt kết quả **PASS**.
