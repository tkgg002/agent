# Kế hoạch thực thi: Audit và Điều chỉnh Chiến lược Refactor (Flow-based)

Kế hoạch thực thi này nhằm rà soát lại kết quả Phase 1 đã làm và định hình lại cách triển khai các Phase tiếp theo theo đúng định hướng "Flow-based".

## Các giai đoạn thực hiện (Phases):

### Phase 1: Audit & Khôi phục sự gắn kết của Recon Module (Giai đoạn 1 đã làm)
- Rà soát các file trong `internal/service/recon/` đã được tách ở các turn trước:
  - `recon_heal.go` (6 file)
  - `provisioning_orchestrator.go` (6 file)
  - `recon_tier_a.go` (5 file)
  - `recon_engine.go` (3 file)
  - `recon_handler.go` (3 file)
  - `scan_handler.go` (3 file)
- **Đánh giá**:
  - `recon_tier_a.go` bị băm quá nhỏ (file chính chỉ còn 271 bytes, tách helpers, lock, prune, run). Nên gộp lại thành tối đa 2 file: 1 file chứa toàn bộ luồng chạy và lock (`recon_tier_a.go`), 1 file chứa các struct/models/prune.
  - `recon_heal.go` bị tách thành 6 file. Nên gộp lại thành tối đa 2-3 file (luồng xử lý lõi vs model/utils phụ trợ).
  - `provisioning_orchestrator.go` bị tách thành 6 file. Nên gộp lại thành tối đa 2-3 file.
- **Hành động**: Đề xuất kế hoạch gom các file này về cấu trúc tinh gọn hơn để giữ tính liên tục của luồng code.

### Phase 2: Áp dụng Quy trình Mới cho Master & Shadow Module (Giai đoạn 2 sắp làm)
- File trọng điểm: `transmuter.go` (903 dòng).
- **Thiết kế mới**: Không tách thành `transmuter_run.go` và `transmuter_extract.go` vì chúng cùng thuộc luồng xử lý transmute dữ liệu shadow -> master.
- Thay vào đó, giữ nguyên luồng chạy chính trong `transmuter.go` (khoảng 600 dòng bao gồm `Run`, `processBatch`, `extractColumns`, `upsertMaster`).
- Chỉ tách 2 phần phụ trợ:
  1. `transmuter_state.go`: Chứa các hàm ghi nhận/persist trạng thái runtime (`markRuntimeSuccess`, `markRuntimeFailure`, `markRuntimeSkipped`, `persistRuntimeState`).
  2. `transmuter_utils.go`: Chứa các helper độc lập về kiểu dữ liệu và SQL (`gjsonValueToGo`, `unwrapMongoExtJSON`, `mongoNumberToGo`, `coerceForColumn`, `epochToTime`, `deterministicGpayID`, các hàm quote SQL).
- Kết quả: Đảm bảo luồng chạy chính cực kỳ liền mạch trong 1 file, không bị "băm nhỏ".

### Phase 3: Biên dịch & Kiểm thử xác thực
- Chạy biên dịch toàn bộ dự án (`go build ./...`) để đảm bảo không lỗi cú pháp hoặc import chéo.
- Chạy toàn bộ bộ test suite (`go test ./...`) để chứng minh tính đúng đắn.
