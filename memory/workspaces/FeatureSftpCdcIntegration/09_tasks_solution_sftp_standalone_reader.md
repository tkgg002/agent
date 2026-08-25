# 09_tasks_solution_sftp_snapshot_standalone_reader.md

## Hồ sơ Giải pháp Kỹ thuật: SFTP Snapshot bằng Standalone Kafka Reader

### 1. Phân tích Nguyên nhân & Lỗ hổng
- **Vấn đề**: Khi kích hoạt Snapshot cho nguồn SFTP (`testsftp21`), `snapshot_runner_handler.go` gọi `client.OffsetCommit` để reset offset của consumer group `cdc-worker-group-local` về 0.
- **Lỗi thực tế**: Kafka Broker trả về lỗi `[25] Unknown Member ID: the member id is not in the current generation`.
- **Lý do**: Kafka Protocol không cho phép một client bên ngoài commit offset cho Consumer Group đang ở trạng thái `ACTIVE` (đang có Worker instance kết nối). Do offset không được reset, Worker không nạp lại thông điệp từ offset 0, dẫn đến snapshot báo `rows_processed = 0`.

---

### 2. Thiết kế Phương án Tối ưu (The Single Best Approach)
Thay vì cố gắng reset Consumer Group Offset trên Kafka Broker (vốn bị Kafka cấm khi group đang ACTIVE), `SnapshotRunner` cho SFTP sẽ:
1. Tạo một **standalone `kafka.Reader`** kết nối trực tiếp vào partition 0 của SFTP topic (không dùng `GroupID`, tránh va chạm với consumer group).
2. Gọi `sftpReader.SetOffset(0)` để nhảy về offset 0 của topic.
3. Đọc lần lượt các thông điệp từ offset 0 đến cuối topic (trong timeout window 30s).
4. Chuyển dữ liệu qua `r.eventHandler.HandleRaw(scopedCtx, topic, msg.Value)` với context mang `shadow_binding_id`.
5. Gọi `r.eventHandler.FlushBatchBuffer(ctx)` để ghi nhận toàn bộ bản ghi xuống PostgreSQL shadow database (`shadow_testsftp21.reconcile`).
6. Lấy số bản ghi thực tế đã lưu (`rowsProcessed`) cập nhật vào `snapshot_progress` với `status = 'done'`.

---

### 3. Chi tiết Mã nguồn Đề xuất cho File `internal/handler/orchestration/snapshot_runner_handler.go`

Trong hàm `runSnapshot` tại nhánh `if isSFTP`:
```go
		// SFTP snapshot: Standalone kafka.Reader reads partition 0 from offset 0 to end of topic
		// and processes messages via eventHandler + FlushBatchBuffer for reliable persistence.
		sftpReader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:   r.kafkaBrokers,
			Topic:     topic,
			Partition: 0,
			MaxBytes:  10e6,
		})
		defer sftpReader.Close()

		if errSeek := sftpReader.SetOffset(0); errSeek != nil {
			r.logger.Warn("sftp snapshot: failed to seek to offset 0", zap.String("topic", topic), zap.Error(errSeek))
		}

		sftpReadCtx, sftpCancel := context.WithTimeout(ctx, 30*time.Second)
		defer sftpCancel()

		readMsgCount := 0
		scopedCtx := ctx
		if p.ShadowBindingID > 0 {
			scopedCtx = handlershadow.WithBindingScope(ctx, p.ShadowBindingID)
		}

		for {
			msg, errMsg := sftpReader.ReadMessage(sftpReadCtx)
			if errMsg != nil {
				break
			}
			readMsgCount++
			if r.eventHandler != nil {
				_, _ = r.eventHandler.HandleRaw(scopedCtx, topic, msg.Value)
			}
		}

		rowsProcessed := 0
		if r.eventHandler != nil {
			if flushed, errFlush := r.eventHandler.FlushBatchBuffer(ctx); errFlush == nil {
				rowsProcessed = flushed
			} else {
				r.logger.Warn("sftp snapshot: FlushBatchBuffer error", zap.Error(errFlush))
			}
		}

		r.logger.Info("sftp snapshot completed via standalone reader",
			zap.String("topic", topic),
			zap.Int("messages_read", readMsgCount),
			zap.Int("rows_processed", rowsProcessed),
		)

		var bindingArg any
		if p.ShadowBindingID > 0 {
			bindingArg = p.ShadowBindingID
		}
		err = r.db.Exec(`
			INSERT INTO cdc_system.snapshot_progress
				(source_object_id, shadow_binding_id, status, trace_id, started_at, updated_at, finished_at, rows_processed)
			VALUES (?, ?, 'done', ?, NOW(), NOW(), NOW(), ?)
		`, p.SourceObjectID, bindingArg, p.TraceID, rowsProcessed).Error
		if err != nil {
			r.logger.Error("failed to insert sftp snapshot progress into database", zap.Error(err))
		}

		return nil
```

---

### 4. Kế hoạch Kiểm thử (Verification Plan)
1. Biên dịch dự án: `go build ./cmd/worker`
2. Chạy test suite: `go test ./internal/handler/orchestration/... ./internal/handler/shadow/...`
