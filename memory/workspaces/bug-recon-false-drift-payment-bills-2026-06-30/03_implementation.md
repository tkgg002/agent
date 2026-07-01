# Technical Design: Fix False Drift on Recon payment_bills / Thiết kế Kỹ thuật: Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills

## 1. Chi tiết Thay đổi Kỹ thuật (Technical Implementation Details)

### A. Cập nhật Recon Dest Agent
* **File**: `internal/service/recon/recon_dest_query.go`
  * Thay đổi signature các hàm:
    * `CountInWindow(ctx context.Context, tableName, timestampField string, tLo, tHi time.Time) (int64, error)`
    * `BucketCounts(ctx context.Context, tableName, pkColumn, timestampField string, tLo, tHi time.Time) (map[int64]BucketStat, error)`
    * `ListIDTsInWindow(ctx context.Context, tableName, pkColumn, timestampField string, tLo, tHi time.Time) ([]IDTs, error)`
    * `MaxWindowTs(ctx context.Context, tableName, timestampField string) (time.Time, error)`
  * Logic:
    * Nếu `timestampField` trống hoặc là `_source_ts`, truy vấn sử dụng cột `_source_ts` kiểu `bigint` (milliseconds) hoặc tìm `MAX("_source_ts")`.
    * Ngược lại, nếu là domain timestamp (ví dụ `lastUpdatedAt`), truy vấn sử dụng cột đó kiểu `timestamp`/`timestamptz`. Giá trị thời gian được convert sang epoch milliseconds đầu giờ để khớp với định dạng của Source MongoDB. Đối với `MaxWindowTs`, thực hiện tìm `MAX(column)` động và parse qua `parsePostgresTimestamp` bằng database/sql `Rows` scan để tránh reflection panic của GORM.
* **File**: `internal/service/recon/recon_dest_hash.go`
  * Thay đổi signature các hàm:
    * `HashWindow(ctx context.Context, tableName, pkColumn, timestampField string, tLo, tHi time.Time) (*WindowResult, error)`
    * `BucketHash(ctx context.Context, tableName, pkColumn, timestampField string) (*BucketHashResult, error)`
  * Logic tương tự, tính toán `hashIDPlusTsMs` dựa trên cột `timestampField` tương ứng.
* **File**: `internal/service/recon/recon_dest_legacy.go`
  * Điều chỉnh signature để giữ tương thích ngược với legacy code.

### B. Cập nhật logic các Tier đối soát và Smoke Test
* **File**: `internal/service/recon/recon_tier_a.go` (Tier 1: Source vs Shadow)
  * Lấy `resolvedTS` từ cấu hình registry (`entry.TimestampField`).
  * Truyền `resolvedTS` vào các hàm gọi `destAgent` (bao gồm `CountInWindow`, `BucketCounts`, `HashWindow`, `ListIDTsInWindow`, và `MaxWindowTs`) để cả hai bên Source (Mongo) và Destination (Postgres Shadow) cùng truy vấn dữ liệu trên cùng một hệ quy chiếu thời gian thực tế.
* **File**: `internal/service/recon/recon_tier_b.go` (Tier 2: Shadow vs Master)
  * Truyền cứng `_source_ts` vào các hàm gọi `destAgent` và `masterAgent` (bao gồm `MaxWindowTs`, `BucketCounts`, `ListIDTsInWindow`) để so sánh CDC stream time (vì cả hai đều nằm trong Postgres).
* **File**: `internal/service/recon/recon_smoke.go`
  * Truyền cứng `_source_ts` vào các hàm gọi `MaxWindowTs` của `destAgent` và `masterAgent`.

## 2. Quản lý Rủi ro & Giải pháp Phòng ngừa (Risk Mitigation)
* **Sql Injection**: Cột `timestampField` được truyền động vào câu SQL raw. Để tránh nguy cơ SQL Injection, toàn bộ các định danh cột và bảng đều được kiểm tra nghiêm ngặt qua hàm validate `validateIdent` và bọc trong các hàm quote an toàn (`quoteIdent`, `quoteRelation`).
* **Hiệu năng Query**: Các truy vấn theo domain timestamp đòi hỏi PostgreSQL phải quét bảng. Để tối ưu hiệu năng, hệ thống tận dụng các partial index đã được thiết lập sẵn trên shadow database.
