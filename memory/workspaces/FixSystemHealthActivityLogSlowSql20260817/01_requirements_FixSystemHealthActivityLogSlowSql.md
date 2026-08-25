# 01_requirements_FixSystemHealthActivityLogSlowSql.md

## 1. Bối cảnh & Hiện trạng
- **Vị trí phát sinh:** `internal/infra/observability/system_health_queries.go:126`
- **Log cảnh báo:**
  ```text
  [201.150ms] [rows:10] SELECT * FROM "cdc_activity_log" WHERE created_at > '2026-08-16 11:19:44.078' AND created_at <= '2026-08-17 11:19:44.078' ORDER BY created_at DESC LIMIT 10
  ```
- **Ngưỡng cảnh báo:** SLOW SQL >= 200ms. Thời gian thực thi thực tế: `201.150ms`.

## 2. Yêu cầu (Specs)
1. **Loại bỏ `SELECT *`:** Chỉ truy vấn các cột cần thiết cho wire format snapshot (`started_at`, `operation`, `target_table`, `status`, `details`).
2. **Chuyển sang Raw SQL có tham số hóa:** Sử dụng `c.db.Raw(...)` với schema rõ ràng `cdc_system.cdc_activity_log` để loại bỏ overhead của GORM ORM builder và reflection.
3. **Hiệu năng mục tiêu:** Thời gian thực thi câu lệnh SQL giảm xuống `< 10ms` (không vượt ngưỡng cảnh báo 200ms).
4. **Bảo toàn giao diện & dữ liệu trả về:** Output `[]map[string]any` giữ nguyên các keys: `time`, `operation`, `table`, `status`, `details`.
