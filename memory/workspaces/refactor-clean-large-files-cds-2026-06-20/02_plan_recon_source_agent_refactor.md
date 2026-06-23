# 02_plan_recon_source_agent_refactor

Roadmap thực thi tái cấu trúc và phân tách file `recon_source_agent.go`.

## Lộ trình các giai đoạn (Phases)

### Phase 1: Lập kế hoạch & Thiết kế Giải pháp (Hiện tại)
- Khảo sát mã nguồn, phân tích các khối logic chính.
- Tạo bộ tài liệu yêu cầu, plan, thiết kế, danh sách task và giải pháp code chi tiết trong workspace.
- Trình bày giải pháp cho User phê duyệt.

### Phase 2: Triển khai chia tách File (Chờ duyệt)
- Tạo 5 file helper mới trong package `recon`:
  - `recon_models.go`
  - `recon_hash.go`
  - `recon_query.go`
  - `recon_stream.go`
  - `recon_legacy.go`
- Rút gọn file gốc `recon_source_agent.go`.
- Dọn dẹp imports không dùng trên từng file.

### Phase 3: Biên dịch & Kiểm thử (Verification)
- Biên dịch dự án bằng `go build ./...` để tìm lỗi cú pháp hoặc khai báo trùng lặp.
- Chạy unit tests của package `recon` bằng `go test -v ./internal/service/recon/...`.
- Chạy unit tests toàn bộ project `go test ./...`.

### Phase 4: Quét bảo mật & Báo cáo
- Thực hiện rà soát bảo mật tĩnh, tạo `report_security_recon_refactor.md`.
- Đếm số dòng code thay đổi thực tế và tạo báo cáo `report_recon_source_agent_refactor.md`.
- Audit lại toàn bộ quy trình so với workspace.
