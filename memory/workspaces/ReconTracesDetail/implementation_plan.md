# Kế hoạch Triển khai Tracing Chi tiết & Tối ưu hóa cho Đối soát dữ liệu

## 1. Phân tích hiện trạng & Nguyên nhân gốc rễ
Hiện tại, khi chạy task đối soát (`HandleReconCheck`), chúng ta gặp 2 vấn đề lớn:
1. **Trace trống (Thiếu child spans):** Trace chỉ hiển thị span cha `cdc.recon.check` ở `cdc-worker` mà không có bất kỳ span con nào. Nguyên nhân do **Bão Span (Span Storm)** trong vòng lặp chia nhỏ window làm tràn hàng đợi OpenTelemetry (`MaxQueueSize = 2048`), khiến các span con được tạo sau đó bị SDK tự động drop.
2. **Hiệu năng chậm (Mất 30 giây cho 5 records):** Khi dải thời gian check là 7 ngày, hệ thống chia nhỏ thành 672 cửa sổ thời gian (15 phút/window) và lặp tuần tự để gọi `HashWindow` cho từng cửa sổ. Cho dù bảng chỉ có 5 records (hầu hết các cửa sổ trống), việc thực hiện tuần tự 672 * 2 = 1344 truy vấn DB vẫn gây trễ tích lũy rất lớn (khoảng 30 giây do độ trễ kết nối/truy vấn mạng).

## 2. Giải pháp kỹ thuật đề xuất

### 2.1. Tối ưu hóa hiệu năng bằng Global Hash Verification & Block Partitioning
Để tránh việc quét tuần tự hàng trăm window trống khi không có drift:
1. **Global Hash Check:** Trước khi chia nhỏ window, thực hiện gọi `HashWindow` trên toàn bộ khoảng thời gian `[lo, hi)` cho cả Source (Mongo) và Destination (Postgres). So sánh tổng số lượng record (`Count`) và XOR hash (`XorHash`). Nếu khớp -> Kết luận không có drift và hoàn thành ngay lập tức (Thời gian thực thi giảm từ 30s xuống còn **< 100ms**). Nếu lệch -> Fallback về chia nhỏ thành các window 15 phút để định vị và drill down chữa lành.
2. **Ngưỡng trần dải thời gian (Threshold Partitioning):** Để phòng ngừa rủi ro Full Table Scan / CPU spike khi dải thời gian quá rộng (ví dụ: check lịch sử 30 ngày), ta đặt ngưỡng trần cho Global Check là **7 ngày**. Nếu dải thời gian `[lo, hi)` > 7 ngày, hệ thống sẽ tự động phân chia thành các Block lớn (tối đa 7 ngày/block) và chạy Global Hash Verification song song hoặc tuần tự cho từng block, thay vì truy vấn một lần duy nhất trên toàn bộ khoảng thời gian cực đại.

### 2.2. Thiết lập Smart Tracing qua Package Observability (Tránh Type Mismatch)
Để tránh rủi ro lệch kiểu dữ liệu (type mismatch) của context key giữa các package khác nhau:
1. Khai báo context key và helper methods tập trung trong package `observability` ([trace_helpers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/observability/trace_helpers.go)):
   ```go
   type contextKey string
   const skipTraceKey contextKey = "recon.skip_window_trace"

   func ContextWithSkipTrace(ctx context.Context) context.Context {
       return context.WithValue(ctx, skipTraceKey, true)
   }

   func IsTraceSkipped(ctx context.Context) bool {
       val, ok := ctx.Value(skipTraceKey).(bool)
       return ok && val
   }
   ```
2. Trong `RunHashWindowCheck` và `RunHashWindowCheckB`, inject cờ bypass qua helper:
   ```go
   ctxLoop = observability.ContextWithSkipTrace(ctxLoop)
   ```
3. Trong `ReconSourceAgent.HashWindow` and `ReconDestAgent.HashWindow`, kiểm tra qua helper:
   ```go
   var span oteltrace.Span
   if !observability.IsTraceSkipped(ctx) {
       ctx, span = observability.ChildSpan(ctx, "recon.source.hash_window", ...)
   }
   ```

### 2.3. Bổ sung các Span con chi tiết cho các tiến trình thực thi
Đảm bảo các bước lớn và quan trọng trong `ReconCore` đều được ghi nhận span:
1. **`cdc.recon.run_hash_window_check_a` / `cdc.recon.run_hash_window_check_b`:** Span cha cho toàn bộ tiến trình Check.
2. **`cdc.recon.verify_global_range`:** Span đo thời gian kiểm tra Global Hash nhanh.
3. **`cdc.recon.window_loop`:** Span bao quanh toàn bộ vòng lặp các window (chỉ chạy khi có drift).
4. **`cdc.recon.drift_drill_down`:** Span được tạo khi phát hiện mismatch ở một window để drill down tìm ID chi tiết.
5. **`cdc.recon.cross_check_shadow`:** Span đo thời gian đối soát chéo với Shadow DB.

### 2.4. Đảm bảo Context Propagation thông suốt
Rà soát và đảm bảo toàn bộ các phương thức gọi xuống DB client (Mongo, GORM) đều kế thừa `ctx` từ parent context mà không bị ngắt quãng.

## 3. Các file sẽ sửa đổi
* `[MODIFY]` [trace_helpers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/observability/trace_helpers.go)
* `[MODIFY]` [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
* `[MODIFY]` [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
* `[MODIFY]` [recon_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go)
* `[MODIFY]` [recon_dest_hash.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go)

## 4. Kế hoạch xác minh (Verification)
1. **Biên dịch:** Chạy `go build ./...` để đảm bảo code build thành công.
2. **Kế hoạch xác minh:** Chạy test suite của recon và master để đảm bảo logic hoạt động chính xác:
   `go test -v ./internal/service/recon/...`
   `go test -v ./internal/handler/recon/...`
   `go test -v ./internal/service/master/...`
