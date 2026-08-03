# 13_analysis_activity_log_fix.md

## Báo cáo Phân tích Chi tiết Lỗi Activity Log & Hướng Xử lý Writer-Only

### 1. Phân tích Nguyên nhân Gốc rễ từ Nguồn Ghi (Writer Root Cause)

Tại sao Log #28232 lại bị `source_database: null`, `source_namespace: null`, `source_table: null`, `shadow_schema: null`, `shadow_table: null`?

- **Phân tích Code Ghi (`batch_buffer.go`):**
  ```go
  targetFQN := schemaName + "." + tableName // => "shadow_testss.schedule_histories"
  logEntry = act.Start("kafka-consumer", targetFQN, "kafka-consumer")
  ```
  `batch_buffer` truyền `targetFQN` vào cột `target_table` của bảng `cdc_activity_log`.

- **Phân tích Code Đọc (`activity_log_read_repo_gorm.go`):**
  ```sql
  LEFT JOIN LATERAL (
      SELECT ... FROM cdc_system.shadow_binding sb
      WHERE sb.shadow_table = al.target_table
  ) sb ON TRUE
  ```
  DB `shadow_binding` lưu `shadow_table = "schedule_histories"` (tên bảng thuần).
  Khi so sánh: `"schedule_histories" = "shadow_testss.schedule_histories"` => **FALSE**.

- **Kết luận:**
  Việc sửa query Read để parse FQN là **workaround sai lầm**. Đúng chuẩn kiến trúc Core Systems là: **`target_table` trong toàn bộ bảng `cdc_activity_log` phải luôn thống nhất là tên bảng thuần (`tableName`, e.g. `"schedule_histories"`)**.
