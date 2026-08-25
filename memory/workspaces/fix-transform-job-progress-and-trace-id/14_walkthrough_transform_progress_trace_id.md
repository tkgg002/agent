# 14_WALKTHROUGH: KẾT QUẢ TRIỂN KHAI & HƯỚNG DẪN KIỂM THỬ

## 1. Tóm tắt kết quả triển khai
Đã hoàn thành 100% các yêu cầu:
1. **Live Progress % & Số Rows (Hoàn thành / Tổng số):**
   - Đếm trước số lượng dòng cần transform/transmute.
   - Cập nhật % tiến độ và số rows hoàn thành sau mỗi chunk/batch.
   - Hiển thị trên UI dạng: `Đang chạy: <hoàn_thành> / <tổng_số> rows (<%>%)` và `Hoàn thành (<hoàn_thành> / <tổng_số>)`.
2. **Compact SigNoz Trace ID:**
   - Tạo/truyền Trace ID xuyên suốt từ Web -> CMS Service -> Worker CDS.
   - Hiển thị 1 icon copy Ant Design `<CopyOutlined />` nhỏ gọn với Tooltip SigNoz Trace ID.
3. **F5 Persistence:**
   - Dữ liệu hoàn thành và Trace ID được lưu vào DB và join LATERAL FQN-safe, khi tải lại trang vẫn hiển thị tức thì.

---

## 2. Kết quả Automated Test
- `centralized-data-service`: `go test ./internal/...` -> PASS.
- `cdc-cms-service`: `go test ./test/...` -> PASS.
- `cdc-cms-web`: `npm run build` -> PASS (1.03s).
