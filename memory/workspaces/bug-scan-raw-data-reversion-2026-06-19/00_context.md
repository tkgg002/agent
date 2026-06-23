# Context: Sửa lỗi Scan Raw Data và Audit Logic Handlers

## 1. Vấn đề hiện tại
- Sau khi thực hiện cấu trúc refactor hexagonal và phân rã các command handler cũ, hàm `HandleScanRawData` trong `scan_handler.go` bị thay đổi logic nghiêm trọng so với commit gốc `c439b9c`.
- **Lỗi cụ thể**:
  - `HandleScanRawData` mới chỉ so sánh các keys lấy được từ JSONB `_raw_data` với các column thực tế trong shadow table. Nó hoàn toàn KHÔNG tự động tạo (auto-create) các mapping rule với status `pending` và `IsActive: false` trong bảng `cdc_system.mapping_rule_v2` như logic gốc của commit `c439b9c`.
  - `HandlePeriodicScan` mới cũng chỉ giả lập message để gọi sang `HandleScanRawData` mới, dẫn đến scheduler chạy định kỳ hoàn toàn vô dụng (không tự động phát hiện schema drift và lưu vào DB).
  - Khoảng 50% tính năng hoạt động sai lệch so với logic cũ do model tự động thay đổi code cũ trong quá trình refactor mà không đối chiếu kỹ.
- **Yêu cầu của User**:
  - Khôi phục logic `HandleScanRawData` và `HandlePeriodicScan` khớp hoàn toàn với logic của commit `c439b9c`.
  - Audit lại toàn bộ logic các function trong các handler để đảm bảo không bị lệch logic cũ.

## 2. Scope & Target
- Target Files:
  - `internal/handler/recon/scan_handler.go`
  - Rà soát các handler khác trong `internal/handler/recon/` và `internal/handler/source/`.
- Tiêu chuẩn:
  - Khôi phục hành vi gốc: quét `_raw_data` -> tìm shadow binding -> tìm existing rules -> đối chiếu -> insert pending rules mới -> publish kết quả NATS.
