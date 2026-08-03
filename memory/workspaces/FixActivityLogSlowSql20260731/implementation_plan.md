# Kế hoạch Tối ưu hóa SLOW SQL Activity Log (cdc-cms-service)

## Mô tả Bài toán & Nguyên nhân Gốc rễ
Hệ thống ghi nhận 3 câu truy vấn SQL bị cảnh báo SLOW SQL (>= 200ms) trên bảng `cdc_system.cdc_activity_log` tại file `internal/infra/persistence/system/activity_log_read_repo_gorm.go`:

1. **Slow Query 1 (`Stats24h` Aggregation - 205.035ms):**
   - *Nguyên nhân:* Bảng `cdc_activity_log` được phân vùng (partition) theo `created_at`. Điều kiện `WHERE started_at > NOW() - INTERVAL '24 hours'` thiếu `created_at` nên PostgreSQL không thể prune partition, buộc phải Seq Scan qua tất cả các partition lịch sử.
2. **Slow Query 2 (`ListActivity` Total Count - 379.618ms):**
   - *Nguyên nhân:* `SELECT COUNT(*) FROM cdc_activity_log al WHERE 1=1` không có điều kiện khoanh vùng thời gian, buộc Postgres quét toàn bộ tuples của tất cả partition để đếm số bản ghi.
3. **Slow Query 3 (`Stats24h` Recent Errors / Enriched List - 333.650ms):**
   - *Nguyên nhân:* Lệnh SQL đang bọc 3 subquery `LEFT JOIN LATERAL` và count subquery cho **TẤT CẢ các dòng `al`** trước khi thực hiện Filter (`WHERE status = 'error'`) và Sort/Limit (`ORDER BY started_at DESC LIMIT 10`).

---

## Giải Pháp Tối Ưu Duy Nhất (Single Best Approach)

### 1. Kích hoạt Partition Pruning & Composite Indexes
- Thêm điều kiện `created_at > NOW() - INTERVAL '24 hours'` (cho Stats 24h) và `created_at > NOW() - INTERVAL '30 days'` (cho Count query mặc định) để Postgres chỉ scan đúng 1-2 partition ngày gần nhất.
- Bổ sung migration file SQL `012_optimize_activity_log_indexes.sql` tạo 2 composite index:
  - `idx_act_created_started_op` ON `cdc_system.cdc_activity_log (created_at DESC, started_at DESC, operation, status)`
  - `idx_act_status_started` ON `cdc_system.cdc_activity_log (status, started_at DESC, created_at DESC)`

### 2. Kỹ thuật Subquery Pagination First (Tối ưu từ 333ms -> < 5ms)
Tái cấu trúc câu truy vấn Enriched List thành mô hình **Subquery / CTE Pagination First**:
- Lọc (`WHERE ...`), Sắp xếp (`ORDER BY started_at DESC`), và Phân trang (`OFFSET ... LIMIT ...`) trực tiếp trên bảng `cdc_activity_log` TRƯỚC trong một derived table `al`.
- Sau khi đã rút gọn chỉ còn đúng **N bản ghi của trang hiện tại** (ví dụ 10 dòng lỗi gần nhất), mới thực hiện `LEFT JOIN LATERAL` với `shadow_binding`, `master_binding`, và `source_object_registry`.
- Nhờ đó, các `LATERAL JOIN` chỉ thi hành đúng N lần thay vì hàng chục ngàn lần.

---

## Proposed Changes

### Database Migration

#### [NEW] `migrations/schema/partitioning/012_optimize_activity_log_indexes.sql`
- Thêm 2 composite index cho `cdc_system.cdc_activity_log` để tăng tốc queries lọc theo status, operation, started_at và created_at.

---

### cdc-cms-service (Go Backend)

#### [MODIFY] `internal/infra/persistence/system/activity_log_read_repo_gorm.go`
- Refactor `Stats24h`: Thêm cờ Partition Pruning `created_at > NOW() - INTERVAL '24 hours'`.
- Refactor `projectionColumns` và `baseFromClause`: Áp dụng chiến lược Subquery Pagination First cho `ListActivity` và `Stats24h` Recent Errors.
- Refactor Count Query trong `ListActivity`: Thêm partition pruning `created_at > NOW() - INTERVAL '30 days'` khi filter rỗng.

---

## Verification Plan

### Automated Tests
- Chạy biên dịch kiểm tra syntax Go: `go build ./cmd/server` tại repo `cdc-cms-service`.

### Manual Verification
- Kiểm tra các endpoint API Activity Log:
  - `GET /api/activity-log`
  - `GET /api/activity-log/stats`
- Kiểm tra log latency xem cả 3 truy vấn SQL đã giảm xuống dưới 50ms (kỳ vọng < 10ms) hay chưa.
