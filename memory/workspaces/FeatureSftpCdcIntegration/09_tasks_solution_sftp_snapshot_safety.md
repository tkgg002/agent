# Kịch bản giải pháp: Bổ sung kiểm tra an toàn nil pointer cho handler SnapshotV2

Tài liệu đặc tả chi tiết phần code cần Muscle chỉnh sửa trong `source_object_actions_handler.go` để tránh lỗi panic nil pointer khi chạy kiểm thử hoặc khi DB chưa khởi tạo.

---

## 1. File `internal/api/source/source_object_actions_handler.go`

Sửa đổi hàm `SnapshotV2` để bổ sung kiểm tra `h.db == nil` và warning log khi không tìm thấy connection code:

```go
	// 1. Phân tích loại nguồn (engine_type) từ database
	if h.db == nil {
		h.logger.Error("database connection not ready in SourceObjectActionsHandler")
		return c.Status(500).JSON(fiber.Map{"error": "db_not_initialized"})
	}

	var soRow struct {
		SourceEngineType   string `gorm:"column:source_engine_type"`
		SourceConnectionID int64  `gorm:"column:source_connection_id"`
	}
	err = h.db.Raw("SELECT source_engine_type, source_connection_id FROM cdc_system.source_object_registry WHERE id = ?", id).Scan(&soRow).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to query source object registry"})
	}

	engine := strings.ToLower(soRow.SourceEngineType)
	isSFTP := engine == "sftp" || engine == "file" || engine == "csv"

	bid := parseBindingIDQuery(c)
	scope, err := h.resolveDispatchScope(c, id)
	if err != nil {
		if ferr, ok := err.(*fiber.Error); ok {
			return c.Status(ferr.Code).JSON(fiber.Map{"error": ferr.Message})
		}
		if errors.Is(err, ports.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "source_object_not_found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "resolve_source_object_scope_failed"})
	}

	user := middleware.GetUsername(c)
	ctx := messaging.WithMetadata(c.UserContext(), user, traceID, c.Get("Idempotency-Key"))

	if isSFTP {
		// ĐỐI VỚI NGUỒN SFTP: Tự động tạo topic Kafka sạch để kích hoạt connector đã chạy
		var connRow struct {
			ConnectionCode string `gorm:"column:connection_code"`
		}
		_ = h.db.Raw("SELECT connection_code FROM cdc_system.connection_registry WHERE id = ?", soRow.SourceConnectionID).Scan(&connRow)

		if connRow.ConnectionCode == "" {
			h.logger.Warn("sftp connection code not found for source object", zap.Int64("id", id))
		}

		var rawConfig map[string]string
		if connRow.ConnectionCode != "" {
			var rawConfigSanitized string
			_ = h.db.Raw("SELECT raw_config_sanitized::text FROM cdc_system.sources WHERE connector_name = ?", connRow.ConnectionCode).Scan(&rawConfigSanitized)
			if rawConfigSanitized != "" {
				_ = json.Unmarshal([]byte(rawConfigSanitized), &rawConfig)
			}
		}
```
