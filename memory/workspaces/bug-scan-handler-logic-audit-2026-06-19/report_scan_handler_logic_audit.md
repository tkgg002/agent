# Report - Sửa Lỗi Logic Scan Handler & Viết Unit Test

## 1. Danh sách các file thay đổi (Files Changed)

| Đường dẫn File | Trạng thái | Số dòng thay đổi (Ước lượng) | Mục đích thay đổi |
|---|---|---|---|
| [scan_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler.go) | MODIFY | ~20 dòng | Sửa logic Unmarshal payload trong `HandleScanArrayFields` và khôi phục backward compatibility trong `HandleScanRawData`. |
| [scan_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler_test.go) | NEW | ~220 dòng | Viết mới bộ unit test bao phủ các kịch bản tương thích ngược và thứ tự unmarshal của Scan Handler. |

Tổng số dòng code thay đổi/thêm mới: **~240 dòng**.

---

## 2. Chi tiết các thay đổi logic

### A. Hàm `HandleScanRawData`
- Khôi phục khả năng xử lý song song cả payload dạng JSON (mới) và payload dạng plain text (tên bảng trực tiếp từ phiên bản cũ).
- Nếu việc parse JSON lỗi, hệ thống tự động fallback dùng toàn bộ payload làm `TargetTable` giúp hệ thống vận hành liên tục mà không bị crash hay skipped ngoài ý muốn.

### B. Hàm `HandleScanArrayFields`
- Sắp xếp lại thứ tự: thực hiện unmarshal tin nhắn NATS nhận được trước để lấy thuộc tính `reply_to` tùy chọn từ payload.
- Fallback về `msg.Reply` nếu payload không chỉ định `reply_to`.
- Trả về tin nhắn phản hồi NATS khi gặp lỗi truy vấn cơ sở dữ liệu thay vì im lặng kết thúc.

---

## 3. Xác thực và Kết quả kiểm thử (Verification & Test Results)

- **Biên dịch**: `go build ./...` thành công không cảnh báo.
- **Unit Test**: Chạy `go test -v ./internal/handler/recon/...` thành công tốt đẹp:
  - `TestHandleScanRawData_BackwardCompatibility`: **PASS**
  - `TestHandleScanArrayFields_ReplyToAndUnmarshalOrder`: **PASS**
- **Toàn bộ Test Suite**: `go test ./...` **PASS** (Không gây lỗi hồi quy).

---

## 4. Rà soát An toàn bảo mật (Security Audit)
- Đã chạy rà soát bảo mật `/security-agent` thủ công.
- Các tham số đầu vào được validate qua hàm `validScanIdent` dùng Regex hạn chế tối đa các ký tự đặc biệt, đảm bảo an toàn tuyệt đối trước lỗ hổng SQL Injection khi thực hiện chuỗi lệnh SQL thô trên bảng đích.
- Verdict: **✅ PASS**
