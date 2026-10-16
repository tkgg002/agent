# 01 Requirements: Canonical MongoDB Extended JSON Scan & Flatten Support

## 1. Bối cảnh & Vấn đề
- Khi scan trường hoặc flatten các collection MongoDB trong hệ thống CDC, các trường lồng chứa kiểu dữ liệu BSON (ví dụ `info.fee: {"$numberInt": "100"}`) bị bộ scan đệ quy sâu vào wrapper `$numberInt`.
- Hậu quả:
  - Sinh ra trường nguồn lồng sai lệch: `info.fee.$numberInt`.
  - Trên CMS UI, tên trường bị cắt lấy phần cuối hiển thị thành `$numberInt`.
  - Khi Transmute sang bảng Master, `gjson.Get` bóc tách chuỗi `"100"` thay vì unwrap ExtJSON, dẫn đến nguy cơ lỗi ép kiểu PostgreSQL hoặc bị gán `NULL`.

## 2. Yêu cầu chi tiết (Specs)
- **REQ-01:** Bổ sung hàm nhận diện Canonical MongoDB Extended JSON `isCanonicalMongoExtJSON` vào `scan_service.go` và `scan_handler.go`.
- **REQ-02:** Cập nhật `flattenJSONWithTypes` trong `scan_service.go` và `scan_handler.go` để chặn đệ quy khi gặp các wrapper ExtJSON (`$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$oid`, `$date`, `$binary`, `$timestamp`, v.v.), gán đúng kiểu PostgreSQL và giữ nguyên tên trường cha (vd `info.fee`).
- **REQ-03:** Cập nhật `InferTypeFromRawData` trong `source_router.go` bổ sung các wrapper còn thiếu (`$numberDouble`, `$binary`, `$timestamp`).
- **REQ-04:** Cập nhật `unwrapExtJSON` trong `transmute/flatten.go` bổ sung `$numberInt` và `$numberDouble`.
- **REQ-05:** Bổ sung unit test bao phủ toàn bộ các wrapper và bảo đảm `go test ./...` cùng `go build` pass 100%.
