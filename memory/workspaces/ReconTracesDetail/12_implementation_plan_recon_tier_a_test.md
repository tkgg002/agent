# Kế hoạch triển khai Unit Tests cho RunHashWindowCheck

Nhiệm vụ: Triển khai file test `internal/service/recon/recon_tier_a_test.go` chứa 3 kịch bản chính của logic `RunHashWindowCheck`.

## 1. Thiết kế cơ chế Mock
Chúng ta cần 3 mock DB riêng biệt thông qua `sqlmock` để mô phỏng 3 vai trò:
1. **Core DB (`rc.db`):** 
   - Quản lý Advisory Lock: `SELECT pg_try_advisory_lock($1)`.
   - Quản lý Run Logs: `INSERT INTO cdc_system.recon_runs ...`, `UPDATE cdc_system.recon_runs SET ...`.
   - Quản lý Lag Logs: `INSERT INTO cdc_system.recon_lag ... ON CONFLICT ...`.
2. **Source DB (`sourceAgent`):**
   - Đóng vai trò Postgres Source.
   - Trả về `MaxWindowTs`: `SELECT MAX("updated_at") FROM "my_table"`.
   - Trả về `HashWindow`: `SELECT id::text AS id, ts AS ts FROM "my_table" WHERE ts >= ? AND ts < ?`.
3. **Dest DB (`destAgent` và `rc.shadowPlane`):**
   - Đóng vai trò Postgres Destination / Shadow DB.
   - Trả về `MaxWindowTs`: `SELECT MAX("updated_at") FROM "my_table"`.
   - Trả về `HashWindow`: `SELECT id::text AS id, _source_ts AS source_ts FROM "my_table" WHERE _source_ts >= ? AND _source_ts < ?` (hoặc `last_updated_at` tùy theo resolve ts field). Ở đây chúng ta sẽ giả lập Postgres source -> target dùng chung cột timestamp `updated_at`.
   - Trả về `ColumnExists` / `CountRows` / `CountDocuments`.

## 2. Chi tiết 3 Test Cases

### 2.1. TestRunHashWindowCheck_GlobalMatch_NoDrift
- **Setup:**
  - Range check: 3 ngày (ví dụ: `2026-07-06T00:00:00Z` đến `2026-07-09T00:00:00Z`).
  - Source DB & Dest DB: MaxWindowTs đều là `2026-07-09T00:00:00Z`.
  - Global Hash Check (Source): Trả về Count = 10, XorHash = 12345.
  - Global Hash Check (Dest): Trả về Count = 10, XorHash = 12345.
- **Kỳ vọng:**
  - `RunHashWindowCheck` trả về Report có Status = "ok", SourceCount = 10, DestCount = 10.
  - Không có window loop con (15 phút) nào được gọi. (sqlmock sẽ báo lỗi nếu có query ngoài mong đợi).

### 2.2. TestRunHashWindowCheck_GlobalMismatch_FallbackToLoop
- **Setup:**
  - Range check: 1 giờ (để số window con nhỏ, ví dụ: 4 window * 15 phút).
  - Source DB & Dest DB: MaxWindowTs đều là `2026-07-09T01:00:00Z`.
  - Global Hash Check (Source): Count = 10, XorHash = 12345.
  - Global Hash Check (Dest): Count = 9, XorHash = 99999 (drift xảy ra).
  - Fallback Loop: Chia thành 4 window con:
    - Window 1, 2, 3: Khớp hash (Count = 2, XorHash = 111).
    - Window 4: Lệch hash (Source: Count = 4, XorHash = 222; Dest: Count = 3, XorHash = 333).
    - Window 4 bị lệch -> Drill down: gọi `ListIDTsInWindow` ở cả 2 bên.
    - Mock `ListIDTsInWindow` và Postgres Shadow check.
- **Kỳ vọng:**
  - `RunHashWindowCheck` trả về Report có Status = "drift", ghi nhận số drifted windows.

### 2.3. TestRunHashWindowCheck_BlockPartitioning
- **Setup:**
  - Range check: 10 ngày. Ngưỡng Global Hash là 7 ngày -> Chia thành 2 block (7 ngày và 3 ngày).
  - Block 1 (7 ngày): Source & Dest đều trả về Count = 50, XorHash = 888.
  - Block 2 (3 ngày): Source & Dest đều trả về Count = 20, XorHash = 999.
- **Kỳ vọng:**
  - Hệ thống kiểm tra Global Hash của từng block trước. Do cả 2 block đều khớp -> Trả về Report Status = "ok", tổng count = 70. Không chạy fallback loop con.

## 3. Các bước triển khai & Xác minh
1. Tạo file `internal/service/recon/recon_tier_a_test.go`.
2. Chạy test lệnh: `go test -v -run TestRunHashWindowCheck ./internal/service/recon/...`.
3. Kiểm tra logs chạy test và ghi nhận kết quả.
