# 01_requirements — Re-Audit Requirements

## R-1: Verification thực tế file:line
- Mọi rating mới PHẢI có 2-3 evidence `file_path:line_number`.
- KHÔNG dựa vào tên file, KHÔNG dựa vào claim cũ.
- Mở file thực, đọc body, confirm logic.

## R-2: Build/Vet/Test còn PASS
- `go build ./...` exit 0 cho cả 3 service Go.
- `go vet ./...` exit 0 (warning cho phép).
- `go test -short -count=1` PASS cho các package có file test.
- Đo lường impact của fix lên runtime: có regression mới không?

## R-3: Detect FAKE claims
- Cross-check mọi claim "PASS" / "exists" trong report cũ với filesystem thực.
- Bất kỳ file claim NEW nào không tồn tại → đánh dấu FAKE.
- Bất kỳ test claim PASS nào không chạy được → đánh dấu FAKE.

## R-4: Composite score recompute
- Tổng điểm tối đa = 16 tiêu chí × 4 = 64.
- Tính lại delta vs audit gốc 35/64 (54.7%).
- Đánh dấu phần trăm đạt target plan 56/64 (87.5%).

## R-5: Gap residual analysis
- Liệt kê các vấn đề CÒN TỒN ĐỌNG sau khi fix.
- Phân loại P0/P1/P2 cho gap residual.
- Có actionable suggestion (không phải reword).

## R-6: Pre-existing failure tracking
- `report_execute_remaining_gaps_2026-05-27.md` §6 ghi nhận 2 pre-existing failure:
  - `TestSanitizeMongoDSN` 4 case.
  - `internal/handler` kafka-go goleak.
- Re-audit phải xác minh 2 failure này đã được fix hay chưa.

## R-7: Governance compliance
- Workspace có đủ 6 file mandatory: 00, 01, 02, 05, 06, 07, 10 + report.
- `05_progress.md` APPEND ONLY.
- §12 — KHÔNG sửa source code trong session re-audit.
