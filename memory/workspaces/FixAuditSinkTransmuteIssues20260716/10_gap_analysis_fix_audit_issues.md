# 10_gap_analysis_fix_audit_issues.md - Phân tích lỗ hổng kiến trúc và rủi ro

Trong quá trình thực thi Phase 0 & Phase 1 của kế hoạch sửa lỗi Sink & Transmute, Brain đã audit chi tiết code thực tế so với logic thiết kế trong plan. 

Dưới đây là các phát hiện quan trọng:

## 1. Thiếu sót/Rủi ro kiến trúc phát hiện được (Medium Risk)

### **Vấn đề: Abort DB transaction do cancel context trong `BatchBuffer.Flush` khi shutdown**
- **Bối cảnh (Trigger):** 
  Khi ứng dụng nhận tín hiệu shutdown (SIGTERM/SIGINT), parent context (`ctx`) bị cancel. Goroutine consume loop kết thúc và gọi `kc.flushAllBatches(ctx)` để ghi nốt toàn bộ dữ liệu đang tích lũy trong memory buffer xuống database.
- **Root Cause:**
  Hàm `BatchBuffer.Flush` sử dụng trực tiếp `bb.ctx` (là context đã bị cancel) để truyền xuống `batchUpsert` và thực hiện các DB operations. GORM/PostgreSQL driver khi phát hiện context đã cancel sẽ lập tức hủy bỏ thực thi câu lệnh SQL và rollback transaction.
- **Hậu quả:**
  Đợt flush cuối cùng khi shutdown sẽ **luôn luôn thất bại** với lỗi `context canceled`. Toàn bộ dữ liệu nghiệp vụ đang nằm trong memory buffer của BatchBuffer tại thời điểm shutdown sẽ không thể ghi xuống DB và bị mất (hoặc gây ra lượng re-deliver và duplicate processing lớn sau khi restart).
- **Giải pháp đề xuất:**
  Trong `Flush()`, thay vì dùng `ctx := bb.ctx`, chúng ta cần tạo một timeout context độc lập từ `context.Background()` để đảm bảo DB writes luôn được thực thi trọn vẹn (best-effort drain) và không bị rollback do cancel signal:
  ```go
  // batch_buffer.go:203
  func (bb *BatchBuffer) Flush() (written int, err error) {
      // Sử dụng background context với timeout 10 giây để đảm bảo ghi DB thành công khi shutdown
      ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
      defer cancel()
      ...
  }
  ```

---

## 2. Đối chiếu chi tiết các hạng mục khác
Tất cả các hạng mục khác đều tuân thủ 100% logic thiết kế của plan:
- **P0-1 & P0-2 (Commit trì hoãn):** Manual commit đã hoạt động thông qua việc thu thập Kafka metadata trong `CDCEvent` -> `UpsertRecord` -> commit sau khi flush thành công từng group.
- **P0-3 (Silent Drops):** Đã bổ sung log và metrics đầy đủ cho cả 4 điểm drop.
- **P0-4 (Recover & Cancel):** Đã bổ sung recover và gọi `cancel()` context để giải phóng connections khi panic.
- **P0-5 (Type Assertions):** Đã chuyển sang switch type cover float64 an toàn.
- **P1-1 (Retry logic):** Đã implement retry backoff và helper `isRetryableDBError`.
- **P1-2 (QueueSubscribe):** Đã chuyển sang QueueSubscribe phân tải NATS.
- **P1-3 & P1-4 (Log & Fallback default):** Đã bổ sung log Warn chi tiết và fallback `DefaultValue` khi validation/transform error cho non-nullable fields.
- **P1-5 (DLQ error swallow):** Đã thay thế ignore error bằng helper `writeFailedSyncLog` có log error và metrics cụ thể.
