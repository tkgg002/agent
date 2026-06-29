# Architectural Decisions: Metadata Cascade Masking Scan Fix

Tài liệu này ghi nhận các quyết định thiết kế kiến trúc được đưa ra trong quá trình giải quyết 6 vấn đề trong workspace `bug-metadata-cascade-masking-scan-fix-2026-06-23`.

## 1. Quyết định 1: Sử dụng LEFT JOIN LATERAL thay thế Subquery lồng trong ListSnapshotProgress
- **Bối cảnh**: Query cũ sử dụng subquery trong `LEFT JOIN shadow_binding sb ON sb.id = COALESCE(sp.shadow_binding_id, (SELECT s.id ...))` để tìm shadow binding id khi null. Điều này làm giảm hiệu năng truy vấn của snapshot-monitor.
- **Quyết định**: Thay thế bằng `LEFT JOIN LATERAL` để tăng hiệu năng và tối ưu hóa logic join:
  ```sql
  LEFT JOIN LATERAL (
      SELECT s.shadow_schema, s.shadow_table
      FROM cdc_system.shadow_binding s
      WHERE (sp.shadow_binding_id IS NOT NULL AND s.id = sp.shadow_binding_id)
         OR (sp.shadow_binding_id IS NULL AND s.source_object_id = sp.source_object_id AND s.is_active = TRUE)
      ORDER BY s.id DESC LIMIT 1
  ) sb ON TRUE
  ```
- **Hệ quả**: Query chạy nhanh hơn, loại bỏ subquery quét đệ quy và đảm bảo hiển thị đúng cột shadow dựa trên `shadow_binding_id` thực tế của progress.

## 2. Quyết định 2: Refactor câu query Master Table lookup sang GORM Named Arguments
- **Bối cảnh**: Hàm `ListMasterTablesByShadowIdentity` dùng positional arguments lồng nhau phức tạp dễ dẫn đến lệch số lượng biến truyền vào khi có thay đổi (dư argument `shadowConnectionKey` ở cuối làm lỗi query).
- **Quyết định**: Chuyển đổi toàn bộ query sang **GORM Named Arguments** (`@arg_name`) truyền qua `map[string]interface{}`.
- **Hệ quả**: Giảm số lượng đối số truyền vào từ 7 xuống còn 4 tham số duy nhất, loại bỏ hoàn toàn rủi ro lệch parameter khi query được bảo trì hoặc mở rộng trong tương lai.

## 3. Quyết định 3: Thay đổi cơ chế xử lý Empty Table Scan
- **Bối cảnh**: Hàm `ScanFieldsDebezium` ném lỗi khi table rỗng làm hỏng log status của command scan fields (chuyển sang "error").
- **Quyết định**: Thay vì ném error làm hỏng status log, hàm sẽ trả về `0, 0, nil` (thành công với 0 fields) và ghi nhận log info.
- **Hệ quả**: Command scan fields kết thúc thành công, log gốc chuyển sang `"success"`, từ đó giúp Frontend nhận diện đúng và tự động dừng polling trạng thái thành công.

## 4. Quyết định 4: Phê duyệt và Đồng bộ DDL Shadow thủ công thông qua UI
- **Bối cảnh**: Khi Operator cập nhật datatype hoặc chuyển một trường sang mã hoá, cấu hình shadow schema sẽ thay đổi. Ban đầu, kế hoạch là tự động gửi tin NATS để alter type cột shadow table DB. Tuy nhiên, trên môi trường Production với bảng dữ liệu lớn (hàng trăm triệu dòng), việc tự động chạy DDL ALTER TABLE qua NATS rất rủi ro, dễ gây lock table và ảnh hưởng hoạt động hệ thống.
- **Quyết định**: 
  1. Loại bỏ logic tự động phát tin NATS DDL từ cdc-cms-service khi lưu mapping rule.
  2. Giao diện Frontend (`cdc-cms-web`) sẽ hiển thị Alert banner cảnh báo Schema Drift khi phát hiện lệch datatype và tích hợp nút bấm **"Đồng bộ Shadow DB ngay"** để Operator chủ động kích hoạt (thực thi API `/create-default-columns` đồng bộ thủ công vào thời điểm tải hệ thống thấp).
- **Hệ quả**: Đảm bảo an toàn vận hành 100%, tránh rủi ro tự động chạy DDL gây downtime hệ thống.

