# 01_requirements_activity_log_fix.md

## Yêu cầu Bài toán: Sửa Lỗi Logging Activity Log (Tối giản & Đúng Gốc Rễ)

### Tuyên bố Tôn chỉ Kỷ luật
1. **Simplicity First, Minimal Impact (Rule #12):** KHÔNG bịa đặt hay nhồi nhét bất kỳ trường mới nào (`master_database`, `shadow_database`, v.v.) vào struct `ActivityLogRow` hay DB schema. Hệ thống vốn không có `shadow_database` hay `master_database` vẫn hoạt động hoàn hảo.
2. **Giữ nguyên 100% service `cdc-cms-service` (Read Side):** Không sửa 1 dòng code hay query SQL Read nào ở `cdc-cms-service`.

---

### Phân tích Đúng Lỗi Gốc rễ

#### Log #28232 (`kafka-consumer`):
- **Hiện trạng lỗi:** `batch_buffer.go` ghi `target_table = "shadow_testss.schedule_histories"` (FQN).
- **Hậu quả:** Câu SQL Read join `sb.shadow_table = al.target_table` so sánh `"schedule_histories" = "shadow_testss.schedule_histories"` bị **FALSE**, làm `source_database`, `source_namespace`, `source_table`, `shadow_schema`, `shadow_table` trả về `null`.
- **Sửa:** Trong `batch_buffer.go`, đổi `target_table` khi gọi `act.Start(...)` từ `targetFQN` về tên bảng thuần `tableName` (`"schedule_histories"`). Khi đó SQL Read join tự động khớp 100% mà không sửa 1 dòng query nào!

#### Log #28233 (`transmute`):
- **Hiện trạng:** Log này vốn dĩ ĐÃ CÓ `target_table = "schedule_histories"` (tên bảng thuần) nên **ĐÃ ENRICH ĐỦ METADATA 100%** từ trước (`source_database: "scheduler-service"`, `shadow_schema: "shadow_testss"`, `shadow_table: "schedule_histories"`).
- **Giữ nguyên:** `triggered_by: "kafka-consumer-hook"` (đúng tác nhân Hook tự động của CDC pipeline).
- **Sửa duy nhất:** Loại bỏ trường `"duration_ms": 20` lặp lại dư thừa bên trong JSON string `details` (vì DB đã có cột `duration_ms: 26`). Bổ sung `"correlation_id"` vào `details` để liên vết với Log 1 (#28232).
