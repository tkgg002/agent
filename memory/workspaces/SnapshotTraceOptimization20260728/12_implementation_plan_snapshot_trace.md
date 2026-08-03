# Kế hoạch Triển khai: Tối ưu Trace Tree cho Snapshot V2 (Batch-centric Tracing)

## Bối cảnh
Người dùng nhận định chính xác: "end nó luôn thì traces bằng gì khi snapshot". Nếu kết thúc Parent Span ngay lập tức và Skip Trace toàn bộ bên trong, hệ thống Observability sẽ mù hoàn toàn đối với tiến trình Snapshot 3 triệu records. Cần một cách tiếp cận đúng đắn hơn cho Batch Job Processing trong OpenTelemetry.

## Phân tích (Tại sao Trace có thể "die"?)
1. **Parent Span kéo dài 3 tiếng**: Về mặt SDK, một Span mở trong bộ nhớ chỉ tốn vài trăm bytes. Nó KHÔNG làm crash ứng dụng. Nhược điểm duy nhất là nó chưa được đẩy lên Collector cho đến khi kết thúc.
2. **Child Span bùng nổ**: Hàm `r.eventHandler.HandleRaw` được gọi *cho từng record*. Nếu snapshot 3 triệu record, nó sẽ tạo ra **3 triệu Span con** trong một khoảng thời gian ngắn. ĐÂY chính là nguyên nhân làm tràn bộ nhớ Worker và làm nghẽn/timeout OTel Collector.

## Giải pháp: Batch-centric Tracing (Tracing theo lô)
Thay vì Trace từng record, chúng ta sẽ Trace theo từng Batch (Lô).
1. **Parent Span**: Giữ Parent Span `nats.SnapshotV2Runner` mở xuyên suốt toàn bộ quá trình `runSnapshot` để gom nhóm toàn bộ Batch.
2. **Batch Span**: Trong vòng lặp `for` của `runSnapshot` (mỗi vòng lặp xử lý 1 batch), tạo một `batchSpan` để đo lường thời gian Query DB và đẩy dữ liệu của 1 Batch.
3. **Record Span (Skip)**: Áp dụng cờ `ContextWithSkipTrace` **chỉ riêng cho các lệnh gọi `HandleRaw`** bên trong lô đó. Điều này triệt tiêu 3 triệu Span rác, nhưng vẫn giữ được 3000 Batch Span (giả sử batch size = 1000) đổ về SigNoz liên tục.

## Các file sẽ thay đổi

### 1. `snapshot_runner_handler.go` (Hàm khởi tạo NATS)
Phục hồi lại `defer span.End()` để Parent Span ôm trọn vòng đời của Snapshot Job. Không gọi SkipTrace ở đây nữa.

#### [MODIFY] `snapshot_runner_handler.go` (line ~115)
```go
	go func(p snapshotV2Payload, jobID string, header nats.Header) {
		ctx := observability.ExtractNATSHeader(context.Background(), header)
		ctx, span := observability.ChildSpan(ctx, "nats.SnapshotV2Runner")
		defer span.End()

		if sc := span.SpanContext(); sc.IsValid() {
			p.TraceID = sc.TraceID().String()
		}

		if err := r.runSnapshot(ctx, p, jobID); err != nil {
            // ...
```

### 2. `snapshot_runner_handler.go` (Hàm `runSnapshot` - Vòng lặp)
Tạo `batchSpan` ở đầu mỗi vòng lặp `for`, và truyền `skipCtx` vào `HandleRaw`.

#### [MODIFY] `snapshot_runner_handler.go` (line ~424)
```go
	for {
		if isPaused.Load() {
            // ...
			return nil
		}
		batchNum++
        
        // --- 1. KHỞI TẠO BATCH SPAN ---
        batchCtx, batchSpan := observability.ChildSpan(ctx, "snapshot.batch", 
            attribute.Int("batch_num", batchNum),
            attribute.Int("batch_size", p.BatchSize),
        )
        // Dùng skipCtx để chặn HandleRaw sinh ra 3 triệu Span con rác
        skipCtx := observability.ContextWithSkipTrace(batchCtx)

		type record struct {
            // ...
```

Và gọi `batchSpan.End()` một cách an toàn bằng cách bọc logic xử lý batch vào một hàm ẩn danh `func() error`, hoặc gọi thủ công trước mỗi lệnh `return/break`.
Cách an toàn nhất để tránh rò rỉ Span là bọc nội dung của vòng lặp `for` vào một hàm ẩn danh.

## Open Questions / Feedback
Anh đánh giá phương án **Batch-centric Tracing** này thế nào? 
- Nếu snapshot 3 triệu dòng (batch size 1000) => Sẽ có 1 Parent Span, và 3000 Batch Spans. Hệ thống OTel Collector hoàn toàn xử lý mượt mà lượng Span này (chỉ ~16 Span/giây), SigNoz sẽ update real-time tiến độ qua từng Batch. Tránh được 3 triệu Span rác.
- Anh vui lòng cho ý kiến (Approve) để em tiến hành apply code nhé.
