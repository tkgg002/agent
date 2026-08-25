# Kịch bản giải pháp: Sửa đổi và chuẩn hóa ghi nhận Snapshot Progress cho SFTP tại cdc-worker

Tài liệu đặc tả chi tiết mã nguồn cần chỉnh sửa trong `snapshot_runner_handler.go` để sửa lỗi ghi nhận trạng thái tiến độ snapshot của nguồn SFTP.

---

## 1. Vấn đề phát hiện qua Audit (QC Findings)
- **Sai trạng thái (Invalid Status):** Đoạn mã hiện tại đang cập nhật status thành `'completed'`. Tuy nhiên, ràng buộc CHECK constraint của bảng `cdc_system.snapshot_progress` chỉ chấp nhận các giá trị `('running', 'done', 'error', 'cancelled', 'paused')`. Từ khóa `'completed'` sẽ gây lỗi vi phạm ràng buộc tại Database.
- **Sai tên cột (Invalid Column):** Đoạn mã hiện tại ghi nhận cột `completed_at`, nhưng thực tế cột trong cơ sở dữ liệu tên là `finished_at`. Dẫn đến lỗi `column completed_at does not exist`.
- **Thiếu bản ghi gốc (Missing Record):** Vì logic SFTP rẽ nhánh và thoát sớm, hàm `acquireLockOrResume` (nơi tạo mới bản ghi `snapshot_progress` với trạng thái `'running'`) bị bỏ qua. Việc này khiến Database hoàn toàn không có bản ghi tiến độ nào cho SFTP. Khi đó, giao diện UI của người dùng không thể nhận diện được tiến trình snapshot đã chạy hay hoàn thành.

---

## 2. File cần sửa đổi: `internal/handler/orchestration/snapshot_runner_handler.go`

Thay thế đoạn logic cập nhật progress cũ bằng câu lệnh `INSERT` trực tiếp bản ghi có trạng thái `'done'` và cột `finished_at` chuẩn xác:

```go
		// Ghi nhận bản ghi snapshot_progress trạng thái 'done' trực tiếp cho SFTP
		var bindingArg any
		if p.ShadowBindingID > 0 {
			bindingArg = p.ShadowBindingID
		}
		err = r.db.Exec(`
			INSERT INTO cdc_system.snapshot_progress
				(source_object_id, shadow_binding_id, status, trace_id, started_at, updated_at, finished_at, rows_processed)
			VALUES (?, ?, 'done', ?, NOW(), NOW(), NOW(), 0)
		`, p.SourceObjectID, bindingArg, p.TraceID).Error
		if err != nil {
			r.logger.Error("failed to insert sftp snapshot progress into database", zap.Error(err))
		}
```
