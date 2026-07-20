# Danh sách Tasks Khắc phục 3 Rủi ro High (SINK-H5, TX-H3, TX-H6)

Tài liệu này theo dõi tiến độ các tasks kỹ thuật được phân rã.

---

## 📋 Danh sách Tasks

- [x] **T1: Khắc phục SINK-H5 (Đồng bộ Fallback)**
  - [x] T1.1: Refactor `batch_buffer.go` thu thập danh sách `successfulRecords` thực tế thành công trong loop fallback tuần tự.
  - [x] T1.2: Triển khai gọi `publishTransmuteTrigger` cho `successfulRecords` khi gặp lỗi transient (thoát sớm).
  - [x] T1.3: Cập nhật hàm `Flush()` để chỉ truyền các records thành công vào trigger sau khi fallback kết thúc.
  - [x] T1.4: Viết Unit Test giả lập lỗi bulk-write và lỗi transient nửa chừng để kiểm tra đồng bộ trigger.

- [x] **T2: Khắc phục TX-H3 (OCC Clock Skew Tolerance)**
  - [x] T2.1: Bổ sung cấu hình `clock_skew_tolerance_ms` (mặc định 2000) vào `AppConfig` (nếu cần) hoặc định nghĩa hằng số an toàn trong `transmuter.go`.
  - [x] T2.2: Cập nhật câu lệnh SQL bulk upsert trong `transmuter.go` phần `DO UPDATE WHERE` kết hợp dung sai: `EXCLUDED._source_ts >= master_table._source_ts - tolerance`.
  - [x] T2.3: Viết Unit Test truyền dữ liệu có timestamp nhỏ hơn trong khoảng tolerance và xác nhận ghi đè thành công.

- [x] **T3: Khắc phục TX-H6 (SHA-256 cho Flatten ID)**
  - [x] T3.1: Refactor hàm `deterministicGpayID` trong `transmuter_utils.go` chuyển từ FNV-1a sang SHA-256.
  - [x] T3.2: Viết Unit Test verify tính ổn định (stable) và tính phân tán của hàm SHA-256 mới.
  - [x] T3.3: Chạy thử nghiệm va chạm với dải 10 triệu bản ghi để chứng minh không trùng lặp.
