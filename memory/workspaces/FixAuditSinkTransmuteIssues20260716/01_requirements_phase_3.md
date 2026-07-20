# v1.13.0-phase-3: Giải Quyết Các Rủi Ro Lớn Còn Tồn Đọng (Remaining Risks)

## Yêu cầu chi tiết

### 1. (TX-C3) Silent Rule Drop Validation
- **Vấn đề:** Trong hàm `loadRules()`, các rules mapping bị skip một cách im lặng khi có `TransformFn` không được whitelist hoặc `DataType` không hợp lệ.
- **Yêu cầu:**
  - Định nghĩa Prometheus counter metric mới: `cdc_transmute_rule_dropped_total` với các nhãn: `master_table`, `source_field`, `target_column`, `reason`.
  - In log Warning mức độ cao chi tiết khi có rule bị drop.
  - Tăng counter metrics để cảnh báo sai lệch cấu hình.

### 2. (SINK-H5) Batch Rollback + Sequential Fallback Protection
- **Vấn đề:** Khi một batch ghi shadow bị lỗi và fallback xuống ghi tuần tự từng dòng, nếu gặp lỗi transient DB (mất mạng, deadlock), code hiện tại ghi nhận sai lệch thành Poison Pill và đẩy vô DLQ.
- **Yêu cầu:**
  - Viết helper `isRetryableDBError` check các lỗi transient (lock, network, timeout).
  - Trong sequential fallback: Nếu gặp lỗi transient, lập tức abort và ném lỗi ra ngoài `Flush()`, dừng commit offset để bảo toàn at-least-once.
  - Chỉ ghi DLQ đối với các lỗi vĩnh viễn (constraint violation, data type error).

### 3. (TX-H3) OCC Timestamp Comparison & Clock Skew
- **Vấn đề:** Clock skew giữa các node có thể làm update muộn có timestamp nhỏ hơn update sớm, dẫn đến update muộn bị bỏ qua (drop update hợp lệ).
- **Yêu cầu:**
  - Tài liệu hóa giải pháp kiểm soát clock skew.
  - Đề xuất giải pháp logic versioning (`_version` hoặc `_sequence_id`) hoặc dual-comparison.

### 4. (TX-H6) FNV Hash Collision in Flatten
- **Vấn đề:** Trùng hash FNV-1a trong flatten có xác suất thấp nhưng vẫn có thể xảy ra khi lượng dữ liệu lớn.
- **Yêu cầu:**
  - Tài liệu hóa rủi ro và các giải pháp giảm thiểu (SHA-256 + 64 bits, hoặc Database sequence map table).
