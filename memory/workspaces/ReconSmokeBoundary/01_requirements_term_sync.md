# Yêu cầu: Đồng bộ hóa thuật ngữ Log Reconciliation (01_requirements_term_sync)

## Mục tiêu
* Loại bỏ hoàn toàn các khái niệm "tier" (như tierA, tierB) khỏi log messages trong `recon_tier_a.go` và `recon_tier_b.go`.
* Đồng bộ các nhãn log tương ứng với TypeRecon tương ứng:
  - `RunHashWindowCheck` (Segment A) -> `[hash_window-A]` và `hash_window-A`.
  - `RunBucketHash` (Segment A) -> `[bucket_hash-A]` và `bucket_hash-A`.
  - `RunHashWindowCheckB` (Segment B) -> `[hash_window-B]`.
  - `RunDeepCheckB` (Segment B) -> `[bucket_hash-B]` hoặc `bucket_hash-B`.
  - Các hàm phụ trợ dùng chung cho Segment A (như `resolveSourceAndDestTSFields`) -> `[recon-A]`.
  - Các hàm phụ trợ dùng chung cho Segment B (như `errorReportB`) -> `recon segment B`.

## Tiêu chuẩn Hoàn thành (Definition of Done)
1. Không còn bất kỳ log message nào trong `recon_tier_a.go` và `recon_tier_b.go` chứa chuỗi `tierA` hay `tierB`.
2. Toàn bộ code compile thành công và pass 100% test suite `go test -v ./internal/service/recon/...`.
3. Kiểm tra governance linter thành công.
