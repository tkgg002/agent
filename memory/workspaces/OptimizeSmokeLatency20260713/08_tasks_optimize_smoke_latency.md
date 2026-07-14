# Danh sách task chi tiết - Optimize Smoke Latency

- `[x]` Khởi tạo workspace `OptimizeSmokeLatency20260713` và các tài liệu bắt đầu.
- `[x]` Viết tài liệu thiết kế kỹ thuật / Kế hoạch triển khai của AI `12_implementation_plan_optimize_smoke_latency.md`.
- `[x]` Phân tích chi tiết `recon_engine.go` và cách thức `effectiveLookback` cùng `RunMode` được sử dụng.
- `[x]` Thiết kế phương án tối ưu logic của `effectiveLookback` và `runLookbackCheckB`.
- `[x]` Trình bày thiết kế lên implementation plan và chờ phê duyệt từ User.
- `[x]` Uỷ quyền cho Muscle thực hiện sửa đổi logic trong `internal/service/recon/recon_engine.go` và `internal/service/recon/recon_smoke.go`.
- `[x]` Chạy unit test của `recon` service để verify tính đúng đắn và phòng ngừa regression.
- `[x]` Kiểm tra độ phủ index trên Postgres shadow DB và MongoDB nguồn để đảm bảo `BucketCounts` chạy nhanh.
- `[x]` Chạy script linter quy trình (`verify_governance.py`) để xác nhận tính tuân thủ.
