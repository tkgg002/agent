# Yêu cầu Khắc phục 3 Rủi ro High (SINK-H5, TX-H3, TX-H6)

Tài liệu này định nghĩa các yêu cầu chi tiết để khắc phục 3 rủi ro High còn lại sau đợt audit luồng Sink & Transmute.

---

## 1. Yêu cầu SINK-H5: Đảm bảo Đồng bộ Shadow ↔ Master khi Fallback Tuần tự
*   **Bối cảnh**: Khi bulk write vào Shadow DB bị lỗi, hệ thống rollback transaction và chạy chế độ fallback tuần tự từng dòng.
*   **Vấn đề hiện tại**:
    *   Nếu gặp lỗi transient (mất kết nối) ở giữa chu kỳ fallback, hàm trả về lỗi và thoát sớm. Các bản ghi đã ghi Shadow thành công trước đó **không được kích hoạt trigger transmute sang Master DB** (do trigger chỉ được bắn ở tầng `Flush` khi cả batch thành công). Điều này gây mất đồng bộ dữ liệu giữa Shadow và Master.
    *   Nếu fallback chạy xong (chỉ có permanent error bị đẩy vào DLQ), trigger vẫn được bắn cho toàn bộ danh sách records ban đầu (bao gồm cả record lỗi không được ghi vào Shadow).
*   **Yêu cầu**:
    *   Thu thập danh sách các record thực sự ghi Shadow thành công trong chu kỳ fallback tuần tự (`successfulRecords`).
    *   Nếu gặp lỗi transient thoát sớm: **Bắt buộc** gọi `publishTransmuteTrigger` cho các `successfulRecords` trước khi trả về lỗi.
    *   Nếu hoàn thành fallback: Chỉ truyền `successfulRecords` vào `publishTransmuteTrigger` thay vì truyền toàn bộ batch ban đầu.

---

## 2. Yêu cầu TX-H3: Khắc phục ảnh hưởng của Clock Skew ở Nguồn CDC (OCC Timestamp Comparison)
*   **Bối cảnh**: Master DB sử dụng cơ chế Optimistic Concurrency Control (OCC) so sánh `EXCLUDED._source_ts >= master_table._source_ts` để chỉ ghi nhận các update mới hơn.
*   **Vấn đề hiện tại**:
    *   Sai lệch múi giờ hoặc clock skew nhỏ giữa các node nguồn phát sinh CDC (do không đồng bộ NTP chuẩn) có thể làm bản ghi mới hơn (nhưng có timestamp nhỏ hơn do clock skew) bị **bỏ qua im lặng (silent skip)**.
*   **Yêu cầu**:
    *   Bổ sung cơ chế dung sai thời gian (Time-window Tolerance) cho so sánh OCC.
    *   Cho phép ghi đè nếu: `EXCLUDED._source_ts >= master_table._source_ts - tolerance_ms` (mặc định dung sai là `2000` ms).
    *   Dung sai phải cấu hình được thông qua hệ thống cấu hình (AppConfig) hoặc hằng số an toàn.

---

## 3. Yêu cầu TX-H6: Tránh va chạm khóa chính (gpay_id) trong giải thuật Flatten
*   **Bối cảnh**: Hàm `deterministicGpayID` sử dụng FNV-1a 64-bit để băm `shadowGpayID` + `keySuffix` thành khóa chính `int63` cho các bản ghi con của mảng (flatten).
*   **Vấn đề hiện tại**:
    *   FNV-1a 64-bit dễ va chạm khi quy mô dữ liệu lớn, đặc biệt là với các chuỗi tuần tự ngắn (e.g. index mảng `0, 1, 2...`). Va chạm khóa chính dẫn đến ghi đè mất mát dữ liệu im lặng (Silent Data Overwrite).
*   **Yêu cầu**:
    *   Thay thế thuật toán băm FNV-1a 64-bit bằng **SHA-256** (lấy 8 bytes đầu tiên của hash sum, ép kiểu thành `uint64` và mask bit dấu để giữ số dương `int64`).
    *   Đảm bảo hàm băm mới có phân bố phân tán hoàn hảo, triệt tiêu nguy cơ va chạm do cấu trúc tuần tự.
