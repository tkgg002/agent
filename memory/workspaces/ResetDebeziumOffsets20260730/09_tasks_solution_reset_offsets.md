# Giải Pháp Kỹ Thuật Chi Tiết - Reset Debezium Connector Offsets

## 1. Backend Modifications (`cdc-cms-service`)

### 1.1 `internal/infra/http/kafka_connect.go`
Bổ sung hàm `Stop` và `DeleteOffsets`:
```go
// Stop stops a connector and its tasks. Upstream endpoint: PUT /connectors/:name/stop
func (c *KafkaConnectClient) Stop(ctx context.Context, name string) error {
	path := fmt.Sprintf("/connectors/%s/stop", url.PathEscape(name))
	return c.doJSON(ctx, http.MethodPut, path, nil, nil)
}

// DeleteOffsets resets/deletes the committed offsets for a stopped connector (Kafka Connect 3.5+).
func (c *KafkaConnectClient) DeleteOffsets(ctx context.Context, name string) error {
	path := fmt.Sprintf("/connectors/%s/offsets", url.PathEscape(name))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}
```

### 1.2 `internal/api/source/system_connectors_handler.go`
Cập nhật `ResetOffsets` handler để tự động dừng Connector trước khi xóa offset:
```go
// POST /api/v1/system/connectors/:name/offsets
func (h *SystemConnectorsHandler) ResetOffsets(c *fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	if !connectorNameRE.MatchString(name) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_connector_name"})
	}

	// Kafka Connect 3.5+ đòi hỏi Connector phải ở trạng thái STOPPED trước khi xóa offset (PUT /connectors/:name/stop).
	// Tự động gọi Stop connector trước (nếu connector đang chạy/pause).
	if err := h.client.Stop(c.UserContext(), name); err != nil {
		h.logger.Warn("failed to stop connector before deleting offsets (will attempt delete anyway)",
			zap.String("connector", name), zap.Error(err))
	}

	if err := h.client.DeleteOffsets(c.UserContext(), name); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "delete_offsets_failed", "detail": err.Error()})
	}
	h.logger.Info("connector offsets reset", zap.String("connector", name))
	return c.Status(202).JSON(fiber.Map{"status": "offsets_deleted", "connector": name})
}
```


### 1.3 `internal/router/router.go`
Đăng ký route Destructive trong khối `DESTRUCTIVE ROUTES (OPS-ADMIN ONLY)`:
```go
registerDestructive("/v1/system/connectors/:name/offsets", h.Source.SystemConnectors.ResetOffsets)
```
Hoặc dùng `api.Delete`:
```go
api.Delete("/v1/system/connectors/:name/offsets", append(destructiveChain, h.Source.SystemConnectors.ResetOffsets)...)
```

---

## 2. Frontend Modifications (`cdc-cms-web`)

### 2.1 `src/pages/SourceConnectors.tsx`
- **Mở rộng type**:
  `type MutationOp = 'restart' | 'pause' | 'resume' | 'restartTask' | 'resetOffsets';`

- **Cập nhật mutation handler** trong `SourceConnectors.tsx`:
  Với `p.op === 'resetOffsets'`, gọi HTTP `POST` (hoặc `DELETE`) `/api/v1/system/connectors/${encodeURIComponent(p.connector)}/offsets`.

- **Nút bấm UI**:
  - Thêm nút **Xóa Offset** (nút `Button` màu warning/danger nhẹ hoặc icon `<ClearOutlined />`) tại cột `Actions` trong bảng Connections & Connectors.
  - Khi người dùng click, kích hoạt Modal xác nhận với thông điệp:
    > *Lưu ý: Connector cần ở trạng thái PAUSED hoặc STOPPED trên Kafka Connect trước khi xóa offset.*
