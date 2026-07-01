# Workspace Plan: Fix False Drift on Recon payment_bills

## Proposed Actions
1. **Phân tích và Khảo sát**:
   - Xác định chênh lệch hệ quy chiếu giữa `lastUpdatedAt` và `_source_ts`.
   - Tìm tất cả các file liên quan đến `ReconDestAgent` và các method query/hash.
2. **Cập nhật ReconDestAgent**:
   - `recon_dest_query.go`: Thêm `timestampField` vào `CountInWindow`, `BucketCounts`, và `ListIDTsInWindow`.
   - `recon_dest_hash.go`: Thêm `timestampField` vào `HashWindow` và `BucketHash`.
   - `recon_dest_legacy.go`: Cập nhật legacy wrapper `GetChunkHashes` truyền default `""` cho `BucketHash`.
3. **Cập nhật Recon Tiers**:
   - `recon_tier_a.go`: Chuyển `resolvedTS` (từ `entry.TimestampField`) vào các phương thức `destAgent`.
   - `recon_tier_b.go`: Truyền `_source_ts` vào các phương thức `destAgent` và `masterAgent` để giữ nguyên đối soát CDC timestamp ở Tier 2.
4. **Viết Unit Tests**:
   - Tạo file test `recon_dest_agent_test.go` chứa các test case verify dynamic timestamp field (cả default `_source_ts` lẫn domain `time.Time`).
5. **Kiểm tra và Xác minh**:
   - Chạy `go build ./internal/... ./cmd/... ./pkgs/...` để kiểm tra compile toàn dự án.
   - Chạy `go test -v ./internal/service/recon/...` để đảm bảo test suite pass 100%.
