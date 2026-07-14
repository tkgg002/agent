# 01 – Yêu cầu: Recon Hash Window Scale Fix

## Bối cảnh
Luồng `hash_window` (Cold Lookback 7 ngày) timeout với lỗi `dst hash window: context deadline exceeded`
trên bảng `schedule_histories` khi data thực tế đạt **50–100 triệu records**.

## Root Cause đã xác định
Kiến trúc hiện tại stream **toàn bộ rows về Go** để XOR hash:
- `ReconDestAgent.HashWindow`: SQL cursor → stream N rows → `limiter.Wait()` mỗi row → XOR in Go
- `ReconSourceAgent.HashWindow`: MongoDB cursor → stream N docs → `limiter.Wait()` mỗi doc → XOR in Go

Toán học tại 100M records:
- 1 window 15 phút ≈ 1.4M rows
- Rate limit 5000 rows/s → 280 giây/window >> QueryTimeout 30s → CRASH
- Tăng timeout/rate vẫn không giải quyết: 672 windows × 70s = 13 giờ

## Yêu cầu kỹ thuật

### R1 – SQL Aggregate cho Postgres Destination
Thay toàn bộ streaming loop trong `ReconDestAgent.HashWindow` bằng 1 SQL aggregate query.
Trả về duy nhất `(count, xor_hash)` – không stream rows.

### R2 – Hash Function Cross-Compatibility
Hash function phải cho ra **kết quả giống hệt nhau** giữa:
- Go (source agent – MongoDB streaming)
- PostgreSQL SQL aggregate (dest agent)

Giải pháp: Dùng **MD5 (8 bytes đầu làm uint64)**:
- Go: `crypto/md5` → `binary.BigEndian.Uint64(h[:8])`
- SQL: `('x'||left(md5(concat(pk,'|',ts_rounded)),16))::bit(64)::bigint`

### R3 – Xóa Rate Limiter khỏi Hash Operations
`limiter.Wait()` chỉ phù hợp cho write ops (heal). Phải xóa khỏi `HashWindow` ở cả
source (MongoDB) và dest (Postgres streaming còn lại).

### R4 – BucketHash consistency
`hashIDPlusTsMs` được dùng chung bởi `HashWindow` và `BucketHash`. Sau khi đổi
algorithm sang MD5, cả 2 chiều (source + dest) tự động consistent.

### R5 – Test coverage
Cập nhật unit test `recon_hash_test.go` để verify:
- MD5-based hash vẫn deterministic
- Cross-compatibility (Go vs SQL formula) – verify bằng golden value
- Drift detection vẫn hoạt động (1ms diff → hash khác)

## Không nằm trong scope
- Không thay đổi `ListIDTsInWindow` (drill-down – data nhỏ, rate limit OK)
- Không thay đổi `BucketHash` logic ngoài hash function
- Không thay đổi DB schema hay migration
- Không thay đổi `QueryTimeout` (giữ nguyên 30s – SQL aggregate chạy trong <1s)
