# YÊU CẦU CHI TIẾT (SPECS): SỬA 3 LỖI SOURCE COUNT & CASCADING SELECTOR TRÊN ACTIVITY MANAGER

## 1. BỐI CẢNH & YÊU CẦU TỪ USER
Người dùng phát hiện 3 lỗi trực tiếp khi thao tác với tác vụ `source_count` (Đếm bản ghi ngày tạo) trên trang `/activity-manager`:
- **Lỗi 1:** Tầng dữ liệu (Layer Type) chọn `source` hoặc `shadow` nhưng không hiện danh sách kết nối ở ô "2. Mã kết nối (Connection Code)" (danh sách rỗng).
- **Lỗi 2:** Chọn Tầng dữ liệu `master`, chọn mã kết nối nhưng không hiện Container / Namespace (Schema SQL, Database Mongo, hoặc Thư mục SFTP) đúng đang có ở mã kết nối đó.
- **Lỗi 3:** Có lỗi (`last_error`) khi chạy `source_count` nhưng không hiển thị nội dung lỗi trực quan trên giao diện $\rightarrow$ Cần đưa chi tiết lỗi vào Tooltip/Popover tại chỗ hiển thị lỗi.

---

## 2. PHẠM VI & TIÊU CHUẨN NGHIỆM THU (DEFINITION OF DONE - DOD)

### G1. Truy vết Yêu cầu 1 (Layer Type Source/Shadow hiển thị Connection):
- [x] Khi người dùng chọn Layer `source`, dropdown Connection Code hiển thị đầy đủ các kết nối nguồn trong hệ thống (`role_type = 'source'`, ví dụ: `traitestpbs`, `traitestces`, `walletservice`).
- [x] Khi người dùng chọn Layer `shadow`, dropdown Connection Code hiển thị đầy đủ kết nối shadow (`role_type = 'shadow'`, ví dụ: `default_shadow`).
- [x] Khi người dùng chọn Layer `master`, dropdown Connection Code hiển thị đầy đủ kết nối master (`role_type = 'master'`, ví dụ: `default_master`, `master_2`, `postgres-develop`).
- [x] Backend CMS `cdc-cms-service` cung cấp API trả về đầy đủ các kết nối trong `cdc_system.connection_registry` (không bị lọc cứng `role_type = 'master'`).

### G2. Truy vết Yêu cầu 2 (Lọc đúng Namespace theo Connection được chọn):
- [x] Khi chọn Layer `master` và chọn một Connection Code cụ thể (ví dụ `master_2` hoặc `postgres-develop`):
  - Hệ thống chỉ hiển thị các Container / Namespace thuộc đúng connection đó.
  - Bao gồm:
    1. Các `master_schema` từ `master_binding` có `master_connection_code === selectedConn` (hoặc `master_connection_id === connection.id`).
    2. `default_database` của connection đó từ `connection_registry` (nếu có, vd: `goopay_master_2`, `core-trans-proxy-history-service`).
    3. `default_schema` của connection đó từ `connection_registry` (nếu có, vd: `public`).
  - Tuyệt đối không hiển thị lẫn lộn các schema của connection master khác.
- [x] Khi chọn Layer `source` và chọn một Connection Code:
  - Chỉ hiển thị các Database / Namespace thuộc về chính connection đó (dựa vào `source_connection_code` và `source_db` trong `source_object_registry`, cùng `default_database` của connection).

### G3. Truy vết Yêu cầu 3 (Tooltip Chi tiết Lỗi khi Chạy Source Count):
- [x] Cột "Lỗi gần nhất" trên bảng danh sách của `/activity-manager`:
  - Khi có lỗi (`last_error != null && last_error != ""`), hiển thị Badge/Tag lỗi màu đỏ nổi bật `Tag color="error"` có icon cảnh báo và con trỏ pointer.
  - Khi hover vào, Tooltip hiển thị trọn vẹn thông báo lỗi kỹ thuật với style container rộng rãi (`maxWidth: 500px`), hỗ trợ bẻ dòng (`wordBreak: 'break-word'`, `whiteSpace: 'pre-wrap'`), kèm thông tin thời gian hoặc tiêu đề rõ ràng.
  - Có nút xem nhanh Popover hoặc copy nội dung lỗi để hỗ trợ người dùng copy gửi kỹ thuật/debug.
- [x] Worker Service (`centralized-data-service`):
  - Đảm bảo khi `source_count` runner gặp lỗi (kể cả panic recover trong `server_scheduler.go`), `last_error` luôn được ghi nhận đầy đủ vào database `cdc_system.cdc_worker_schedule`.
  - Khi chạy thành công, `last_error` được reset về `NULL`.
