# 02_plan_activity_log_fix.md

## Kế hoạch Tối giản Tối đa (Simplicity First & Minimal Impact)

### 1. Phân tích Điểm Lỗi Duy Nhất
- **Lỗi ở `batch_buffer.go`:**
  `logEntry = act.Start("kafka-consumer", targetFQN, "kafka-consumer")`
  Do truyền `targetFQN` (`"shadow_testss.schedule_histories"`), câu SQL Read join `sb.shadow_table = al.target_table` ở `cdc-cms-service` bị so sánh `"schedule_histories" = "shadow_testss.schedule_histories"` (FALSE), làm `source_database`, `shadow_schema`, `shadow_table` của Log #28232 bị `null`.
- **Log #28233 (`transmute`):**
  Vốn dĩ ĐÃ CÓ `target_table = "schedule_histories"` (tên bảng thuần) nên đã enrich đủ 100% metadata từ trước.
- **Không thêm trường ảo:** Tuyệt đối KHÔNG thêm `master_database` hay sửa struct `ActivityLogRow`. Hệ thống không cần `shadow_database` hay `master_database` vẫn chạy bình thường.

---

### 2. Hai Thay Đổi Duy Nhất (Chỉ thuộc `centralized-data-service`)

1. **`batch_buffer.go` (và `sinkworker/worker.go`):**
   Đổi `targetFQN` sang `tableName` khi gọi `act.Start("kafka-consumer", tableName, "kafka-consumer")`.
2. **`transmute_handler.go`:**
   Giữ nguyên `triggered_by: "kafka-consumer-hook"`.
   Loại bỏ `"duration_ms"` lặp dư thừa trong JSON `details`, thêm `"correlation_id"`.

### 3. File GIỮ NGUYÊN 100%
- `cdc-cms-service`: **GIỮ NGUYÊN 100%**, không đụng 1 dòng code hay query nào!
