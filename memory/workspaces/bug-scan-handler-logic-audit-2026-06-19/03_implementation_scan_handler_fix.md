# Scan Handler Logic Fix

> Date: 2026-06-19
> Trigger: Lỗi logic unmarshal sai thứ tự trong HandleScanArrayFields và thiếu tương thích ngược trong HandleScanRawData
> Status: ✅ RESOLVED

---

## 1. Symptom
- **HandleScanArrayFields**: Giải mã JSON payload xảy ra sau các bước kiểm tra metadata schema/database, khiến hệ thống không trích xuất được thông tin cấu hình `reply_to` sớm. Khi có lỗi truy vấn cơ sở dữ liệu xảy ra trước đó, tin nhắn phản hồi lỗi bị gửi sai kênh hoặc bị nuốt hoàn toàn.
- **HandleScanRawData**: API cũ hỗ trợ gửi trực tiếp plain text (tên bảng). Khi nâng cấp lên JSON payload, code mới parse JSON lỗi đối với plain text làm gãy khả năng tích hợp của các client cũ (regression).

---

## 2. Iteration Timeline
- **2026-06-19 15:21:00Z**: Phát hiện & Audit logic tổng quát.
- **2026-06-19 15:22:00Z**: Xác lập Giả thuyết 1 (Sắp xếp lại thứ tự unmarshal trong HandleScanArrayFields) và Giả thuyết 2 (Thực hiện try-unmarshal với fallback plain text cho HandleScanRawData).
- **2026-06-19 15:22:30Z**: Xác nhận Root Cause. Thực hiện sửa đổi mã nguồn trong file [scan_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler.go).
- **2026-06-19 15:22:48Z**: Tạo mới tệp unit test [scan_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler_test.go).
- **2026-06-19 15:23:50Z**: Kiểm thử và xác thực thành công các test case.
- **2026-06-19 15:24:10Z**: Rà soát bảo mật qua `/security-agent` (Verdict: PASS).

---

## 3. Root Cause
- **HandleScanArrayFields**: Logic nghiệp vụ (resolve target schema, check table exists) chạy trước lệnh `json.Unmarshal(msg.Data, &payload)`, làm mất thông tin `reply_to` của payload khi xử lý exception từ DB.
- **HandleScanRawData**: Sự thiếu hụt của cơ chế try-parse JSON và cơ chế fallback về plain text payload gốc khi gặp lỗi parse JSON.

---

## 4. Fix
- **scan_handler.go**:
  - Di chuyển block unmarshal lên đầu hàm `HandleScanArrayFields` và gán fallback `replySubject = msg.Reply` nếu không có custom `reply_to`.
  - Bổ sung fallback giải mã thô: gán `payload.TargetTable = string(msg.Data)` nếu `json.Unmarshal` thất bại trong `HandleScanRawData`.
- **scan_handler_test.go**:
  - `TestHandleScanRawData_BackwardCompatibility`: Xác thực xử lý thành công cả JSON payload và plain text payload.
  - `TestHandleScanArrayFields_ReplyToAndUnmarshalOrder`: Xác thực thứ tự unmarshal và phản hồi tin nhắn chính xác về kênh NATS được chỉ định.

---

## 5. Verify
- `go test -v ./internal/handler/recon/...` => **PASS**
- `go test ./...` => **PASS**

---

## 6. Related lessons
- `lessons.md` => "Simplicity First, minimal impact", "Symptom vs cause separation"

---

## 7. Follow-ups
- Không có.
