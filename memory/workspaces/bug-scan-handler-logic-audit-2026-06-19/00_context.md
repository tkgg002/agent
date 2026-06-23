# Context: Audit Logic Toàn Diện & Sửa Lỗi Scan Handlers

## 1. Vấn đề hiện tại
- Sau khi khôi phục logic quét raw data (`HandleScanRawData`) và định kỳ (`HandlePeriodicScan`) ở phiên trước, chúng ta phát hiện thêm một lỗi logic nghiêm trọng trong hàm `HandleScanArrayFields` tại `scan_handler.go`.
- **Lỗi cụ thể trong `HandleScanArrayFields`**:
  - Thứ tự gán `replySubject := msg.Reply` và kiểm tra `payload.ReplyTo` diễn ra trước khi gọi `json.Unmarshal(msg.Data, &payload)`.
  - Điều này làm cho `payload.ReplyTo` luôn bị rỗng khi kiểm tra, khiến tin nhắn phản hồi NATS không thể gửi về custom subject yêu cầu bởi payload.
- **Yêu cầu của User**:
  - Rà soát, audit từng dòng logic của toàn bộ các function trong `scan_handler.go` và các scan handlers khác để phát hiện các lỗi sai lệch logic so với phiên bản gốc ổn định.
  - Sửa đổi các lỗi logic phát hiện được để khôi phục 100% tính năng hoạt động chính xác.

## 2. Scope & Target
- Target Files:
  - `internal/handler/recon/scan_handler.go`
  - Rà soát các logic liên quan đến scan trong cùng package/layer.
- Tiêu chuẩn hoàn thành (DoD):
  - Khắc phục lỗi thứ tự Unmarshal trong `HandleScanArrayFields`.
  - Đảm bảo toàn bộ các logic kiểm tra, Resolve Metadata, chèn MappingRuleV2 và gửi phản hồi NATS trong `scan_handler.go` hoạt động chính xác.
  - Dự án build thành công (`go build ./...`) và toàn bộ test suite pass 100%.
