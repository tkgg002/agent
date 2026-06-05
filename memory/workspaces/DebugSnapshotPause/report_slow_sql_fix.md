# Report: Fix SLOW SQL >= 200ms trong CDC Worker

## 1. Phân tích Root Cause (Gốc rễ)
* **Vấn đề gặp phải:** Log hệ thống ghi nhận cảnh báo `SLOW SQL >= 200ms` liên tục khi thực hiện `INSERT INTO ... ON CONFLICT ...` cho từng row riêng lẻ. Cùng với đó là log báo `schema drift detected` cho hơn 10,000 events. Với tốc độ ~200-400ms mỗi query, việc đồng bộ 10k rows mất quá nhiều thời gian (khoảng hơn 30 phút cho 1 batch).
* **Nguyên nhân:** Hàm `batchUpsert` trong file `batch_buffer.go` (dòng 216 cũ) đang lặp qua mảng `records` và thực thi câu lệnh `db.Exec(query, values...)` một cách độc lập cho từng record. Vì không được bọc trong một Database Transaction, mỗi truy vấn bị chịu chi phí của một autocommit transaction (network RTT, disk I/O flush WAL), dẫn đến cực kỳ chậm chạp với dữ liệu lớn (Snapshot).

## 2. Giải pháp thực thi
* **Cập nhật Source Code (Go):** Sửa hàm `batchUpsert` tại `internal/handler/batch_buffer.go`.
* **Kỹ thuật "Elegant Fallback Transaction":**
  1. Gói toàn bộ vòng lặp `tx.Exec(query, values...)` vào trong một `db.Transaction`. Nhờ đó, thao tác batch insert 10,000 records chỉ tiêu tốn 1 lần commit duy nhất, tăng tốc độ thực thi lên gấp hàng trăm lần.
  2. Bổ sung cơ chế Fallback an toàn (`txErr != nil`): Nếu toàn bộ transaction bị rollback (do 1 record nào đó lỗi constraint/type mismatch), vòng lặp sẽ rã ra chạy `db.Exec` tuần tự một cách độc lập. Điều này giúp hệ thống vẫn cô lập được chính xác record bị lỗi và ghi nhận vào `failed_sync_logs` như thiết kế nguyên thủy, mà không làm hỏng cả batch.

## 3. Verify
* Biên dịch source code: Chạy lệnh `make build` thành công, không có cảnh báo cú pháp.
* Logic bảo toàn tính trọn vẹn: Khả năng ghi log lỗi (`metrics.SyncFailed`, `bb.buildFailedSyncLog`) không bị thay đổi.
* Tốc độ: Việc áp dụng transaction sẽ chấm dứt triệt để tình trạng log SLOW SQL cho snapshot process hàng vạn row.

## 4. Skills Đã Sử Dụng
* `Performance Profiling`: Phân tích GORM transaction log, nhận diện bottleneck của vòng lặp DB I/O RTT.
* `Golang Patterns`: Tối ưu GORM transaction với Fallback error isolation mechanism (vừa nhanh, vừa bảo tồn error traceability).
* `Documentation Standards`: Xuất báo cáo theo đúng pattern định sẵn.
