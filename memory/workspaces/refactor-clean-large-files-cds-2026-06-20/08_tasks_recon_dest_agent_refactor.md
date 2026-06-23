# Checklist Thực thi - Recon Dest Agent Refactor

Dưới đây là các task cụ thể cần Muscle thực thi và Brain kiểm thử sau khi User phê duyệt kế hoạch.

## Task 1: Khởi tạo các file helper mới
- [ ] Tạo file `recon_dest_models.go` và dán định nghĩa config, struct phụ trợ.
- [ ] Tạo file `recon_dest_hash.go` và dán logic băm XOR, xxhash.
- [ ] Tạo file `recon_dest_query.go` và dán logic count/aggregate queries.
- [ ] Tạo file `recon_dest_stream.go` và dán logic list IDs / stream.
- [ ] Tạo file `recon_dest_legacy.go` và dán legacy ChunkHash shim.
- [ ] Tạo file `recon_dest_safety.go` và dán các hàm validate/quote SQL identifier.

## Task 2: Cập nhật file core
- [ ] Ghi đè file core `recon_dest_agent.go` đã rút gọn.
- [ ] Kiểm tra imports, sửa các lỗi compiler (nếu có).

## Task 3: Kiểm thử và Xác minh
- [ ] Chạy biên dịch toàn bộ dự án: `go build ./...`
- [ ] Chạy unit tests package `recon`: `go test -v ./internal/service/recon/...`
- [ ] Chạy unit tests toàn bộ dự án: `go test ./...`

## Task 4: Báo cáo
- [ ] Tạo file báo cáo LOC: `report_recon_dest_agent_refactor.md`
- [ ] Cập nhật kết quả rà soát bảo mật `report_security_recon_refactor.md`
- [ ] Cập nhật nhật ký tiến độ `05_progress.md`
- [ ] Cập nhật active_plans.md và walkthrough của phiên.
